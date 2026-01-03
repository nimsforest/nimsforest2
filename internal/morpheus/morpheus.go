// Package morpheus provides configuration loading for NimsForest cluster deployments.
// It reads node information and cluster registry from well-known paths.
package morpheus

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const (
	// DefaultNodeInfoPath is the default path to the local node configuration file.
	DefaultNodeInfoPath = "/etc/morpheus/node-info.json"

	// DefaultRegistryPath is the default path to the shared cluster registry file.
	DefaultRegistryPath = "/mnt/forest/registry.json"

	// DefaultClusterPort is the default port for NATS cluster communication.
	DefaultClusterPort = 6222
)

// NodeInfoPath returns the path to the node info file.
// Can be overridden with MORPHEUS_NODE_INFO environment variable.
func NodeInfoPath() string {
	if path := os.Getenv("MORPHEUS_NODE_INFO"); path != "" {
		return path
	}
	return DefaultNodeInfoPath
}

// RegistryPath returns the path to the registry file.
// Can be overridden with MORPHEUS_REGISTRY environment variable.
func RegistryPath() string {
	if path := os.Getenv("MORPHEUS_REGISTRY"); path != "" {
		return path
	}
	return DefaultRegistryPath
}

// NodeInfo contains the local node's identity in the forest.
type NodeInfo struct {
	ForestID string `json:"forest_id"` // Cluster/forest identifier
	NodeID   string `json:"node_id"`   // Unique node identifier
}

// Registry contains the shared cluster registry of all nodes.
type Registry struct {
	Nodes map[string][]Node `json:"nodes"` // Map of forest_id to list of nodes
}

// Node represents a single node in the cluster registry.
type Node struct {
	ID       string `json:"id"`             // Unique node identifier
	IP       string `json:"ip"`             // Primary IP (IPv6 preferred, IPv4 fallback)
	IPv6     string `json:"ipv6,omitempty"` // IPv6 address (if available)
	IPv4     string `json:"ipv4,omitempty"` // IPv4 address (if available)
	ForestID string `json:"forest_id"`      // Cluster/forest identifier
}

// GetPreferredIP returns the best IP address to use based on available connectivity.
// Prefers IPv6 if available and hasIPv6Connectivity is true, falls back to IPv4.
func (n *Node) GetPreferredIP(hasIPv6Connectivity bool) string {
	if hasIPv6Connectivity && n.IPv6 != "" {
		return n.IPv6
	}
	if n.IPv4 != "" {
		return n.IPv4
	}
	// Fallback to legacy IP field
	return n.IP
}

// Load reads the local node configuration from the standard path.
// Returns nil if the file doesn't exist or can't be read.
func Load() *NodeInfo {
	return LoadFrom(NodeInfoPath())
}

// LoadFrom reads node configuration from a custom path.
func LoadFrom(path string) *NodeInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[Morpheus] Warning: failed to read node info from %s: %v", path, err)
		}
		return nil
	}

	var info NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		log.Printf("[Morpheus] Warning: failed to parse node info from %s: %v", path, err)
		return nil
	}

	log.Printf("[Morpheus] Loaded node info: forest_id=%s, node_id=%s", info.ForestID, info.NodeID)
	return &info
}

// LoadRegistry reads the cluster registry from the standard path.
func LoadRegistry() (*Registry, error) {
	return LoadRegistryFrom(RegistryPath())
}

// LoadRegistryFrom reads the cluster registry from a custom path.
func LoadRegistryFrom(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{Nodes: make(map[string][]Node)}, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	if reg.Nodes == nil {
		reg.Nodes = make(map[string][]Node)
	}

	return &reg, nil
}

// GetPeers returns the cluster peer addresses for a given forest, excluding the current node.
// Each peer address is formatted as "[IPv6]:port" or "IPv4:port" for NATS cluster connection.
// Defaults to preferring IPv6 connectivity.
func GetPeers(forestID, selfIP string) []string {
	return GetPeersFrom(RegistryPath(), forestID, selfIP, DefaultClusterPort, true)
}

// GetPeersWithConnectivity returns peers using the specified IP connectivity preference.
func GetPeersWithConnectivity(forestID, selfIP string, hasIPv6Connectivity bool) []string {
	return GetPeersFrom(RegistryPath(), forestID, selfIP, DefaultClusterPort, hasIPv6Connectivity)
}

// GetPeersFrom reads peers from a custom registry path.
func GetPeersFrom(registryPath, forestID, selfIP string, clusterPort int, hasIPv6Connectivity bool) []string {
	reg, err := LoadRegistryFrom(registryPath)
	if err != nil {
		log.Printf("[Morpheus] Warning: failed to load registry: %v", err)
		return nil
	}

	nodes, ok := reg.Nodes[forestID]
	if !ok {
		log.Printf("[Morpheus] No nodes found for forest: %s", forestID)
		return nil
	}

	var peers []string
	for _, node := range nodes {
		// Get the preferred IP for this node
		peerIP := node.GetPreferredIP(hasIPv6Connectivity)

		// Skip self (check against all possible IPs)
		if peerIP == selfIP || node.IP == selfIP || node.IPv4 == selfIP || node.IPv6 == selfIP {
			continue
		}

		// Format address based on IP version
		var addr string
		if isIPv6(peerIP) {
			addr = fmt.Sprintf("[%s]:%d", peerIP, clusterPort)
		} else {
			addr = fmt.Sprintf("%s:%d", peerIP, clusterPort)
		}
		peers = append(peers, addr)
	}

	log.Printf("[Morpheus] Found %d peers for forest %s (IPv6 connectivity: %v)", len(peers), forestID, hasIPv6Connectivity)
	return peers
}

// isIPv6 checks if an IP address is IPv6
func isIPv6(ip string) bool {
	// IPv6 addresses contain colons
	for i := 0; i < len(ip); i++ {
		if ip[i] == ':' {
			return true
		}
	}
	return false
}

// RegisterNode adds or updates a node in the registry.
// This is typically called by the deployment system (Morpheus).
func RegisterNode(node Node) error {
	return RegisterNodeTo(RegistryPath(), node)
}

// RegisterNodeTo adds or updates a node in a custom registry file.
func RegisterNodeTo(registryPath string, node Node) error {
	// Use a file lock for concurrent writes
	lockPath := registryPath + ".lock"
	lock := &fileLock{path: lockPath}

	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	// Load existing registry
	reg, err := LoadRegistryFrom(registryPath)
	if err != nil {
		return err
	}

	// Update or add the node
	nodes := reg.Nodes[node.ForestID]
	updated := false
	for i, n := range nodes {
		if n.ID == node.ID {
			nodes[i] = node
			updated = true
			break
		}
	}
	if !updated {
		nodes = append(nodes, node)
	}
	reg.Nodes[node.ForestID] = nodes

	// Write back
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	log.Printf("[Morpheus] Registered node: id=%s, forest=%s, ip=%s, ipv4=%s, ipv6=%s", node.ID, node.ForestID, node.IP, node.IPv4, node.IPv6)
	return nil
}

// UnregisterNode removes a node from the registry.
func UnregisterNode(forestID, nodeID string) error {
	return UnregisterNodeFrom(RegistryPath(), forestID, nodeID)
}

// UnregisterNodeFrom removes a node from a custom registry file.
func UnregisterNodeFrom(registryPath, forestID, nodeID string) error {
	lockPath := registryPath + ".lock"
	lock := &fileLock{path: lockPath}

	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	reg, err := LoadRegistryFrom(registryPath)
	if err != nil {
		return err
	}

	nodes := reg.Nodes[forestID]
	for i, n := range nodes {
		if n.ID == nodeID {
			reg.Nodes[forestID] = append(nodes[:i], nodes[i+1:]...)
			break
		}
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	log.Printf("[Morpheus] Unregistered node: id=%s, forest=%s", nodeID, forestID)
	return nil
}

// Simple file-based lock for registry operations
type fileLock struct {
	path string
	mu   sync.Mutex
}

func (l *fileLock) Lock() error {
	l.mu.Lock()

	// Try to create lock file with exponential backoff
	for i := 0; i < 10; i++ {
		f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.Close()
			return nil
		}
		if !os.IsExist(err) {
			l.mu.Unlock()
			return err
		}
		time.Sleep(time.Duration(1<<i) * 10 * time.Millisecond)
	}

	// Force acquire lock after timeout
	os.Remove(l.path)
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		l.mu.Unlock()
		return err
	}
	f.Close()
	return nil
}

func (l *fileLock) Unlock() {
	os.Remove(l.path)
	l.mu.Unlock()
}

// MustLoad reads the local node configuration and panics if it's missing.
// Use this when Morpheus configuration is required.
func MustLoad() *NodeInfo {
	return MustLoadFrom(NodeInfoPath())
}

// MustLoadFrom reads node configuration from a custom path and panics if missing.
func MustLoadFrom(path string) *NodeInfo {
	info := LoadFrom(path)
	if info == nil {
		log.Fatalf("[Morpheus] FATAL: Node configuration required at %s", path)
	}
	if info.ForestID == "" {
		log.Fatalf("[Morpheus] FATAL: forest_id is required in %s", path)
	}
	if info.NodeID == "" {
		log.Fatalf("[Morpheus] FATAL: node_id is required in %s", path)
	}
	return info
}
