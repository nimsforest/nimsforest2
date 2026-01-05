package viewmodel

import (
	"fmt"
	"runtime"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/shirou/gopsutil/v3/mem"
)

// ClusterSnapshot represents a point-in-time snapshot of the NATS cluster state.
type ClusterSnapshot struct {
	LocalNode   NodeInfo   `json:"local_node"`
	PeerNodes   []NodeInfo `json:"peer_nodes"`
	CapturedAt  time.Time  `json:"captured_at"`
	ClusterName string     `json:"cluster_name"`
}

// NodeInfo represents information about a NATS node.
type NodeInfo struct {
	ID          string    `json:"id"`           // Server ID
	Name        string    `json:"name"`         // Server name
	Host        string    `json:"host"`         // Host address
	Port        int       `json:"port"`         // Client port
	ClusterPort int       `json:"cluster_port"` // Cluster port
	ClusterURL  string    `json:"cluster_url"`  // Cluster route URL
	RAMTotal    uint64    `json:"ram_total"`    // Total RAM
	CPUCores    int       `json:"cpu_cores"`    // CPU cores
	GPUVram     uint64    `json:"gpu_vram"`     // GPU VRAM (0 if none)
	GPUTflops   float64   `json:"gpu_tflops"`   // GPU compute power
	StartTime   time.Time `json:"start_time"`   // When the server started
	IsLocal     bool      `json:"is_local"`     // Whether this is the local node
}

// SubscriptionInfo represents information about a subscription.
type SubscriptionInfo struct {
	Subject  string `json:"subject"`   // Subject pattern
	Queue    string `json:"queue"`     // Queue group (if any)
	NumMsgs  uint64 `json:"num_msgs"`  // Messages delivered
	NumBytes uint64 `json:"num_bytes"` // Bytes delivered
}

// StreamInfo represents information about a JetStream stream.
type StreamInfo struct {
	Name      string   `json:"name"`
	Subjects  []string `json:"subjects"`
	Messages  uint64   `json:"messages"`
	Bytes     uint64   `json:"bytes"`
	Consumers int      `json:"consumers"`
}

// ConsumerInfo represents information about a JetStream consumer.
type ConsumerInfo struct {
	StreamName   string `json:"stream_name"`
	Name         string `json:"name"`
	FilterSubject string `json:"filter_subject"`
	NumPending   uint64 `json:"num_pending"`
	NumAckPending uint64 `json:"num_ack_pending"`
}

// Reader reads cluster state from an embedded NATS server.
type Reader struct {
	server *server.Server
}

// NewReader creates a new Reader for the given NATS server.
func NewReader(ns *server.Server) *Reader {
	return &Reader{
		server: ns,
	}
}

// ReadClusterState reads the current cluster state from the embedded NATS server.
func (r *Reader) ReadClusterState() (*ClusterSnapshot, error) {
	if r.server == nil {
		return nil, fmt.Errorf("no NATS server available")
	}

	snapshot := &ClusterSnapshot{
		CapturedAt:  time.Now(),
		ClusterName: r.getClusterName(),
	}

	// Get local node info
	localNode, err := r.getLocalNodeInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get local node info: %w", err)
	}
	snapshot.LocalNode = localNode

	// Get peer nodes from routes
	peerNodes, err := r.getPeerNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get peer nodes: %w", err)
	}
	snapshot.PeerNodes = peerNodes

	return snapshot, nil
}

// getClusterName returns the cluster name from the server options.
func (r *Reader) getClusterName() string {
	if r.server == nil {
		return ""
	}
	// Try to get cluster name from Varz
	varz, err := r.server.Varz(&server.VarzOptions{})
	if err == nil && varz != nil && varz.Cluster.Name != "" {
		return varz.Cluster.Name
	}
	return "default"
}

// getLocalNodeInfo returns information about the local NATS node.
func (r *Reader) getLocalNodeInfo() (NodeInfo, error) {
	// Use Varz to get server information
	varz, err := r.server.Varz(&server.VarzOptions{})
	if err != nil {
		return NodeInfo{}, fmt.Errorf("failed to get varz: %w", err)
	}

	node := NodeInfo{
		ID:        varz.ID,
		Name:      varz.Name,
		Host:      varz.Host,
		Port:      varz.Port,
		CPUCores:  runtime.NumCPU(),
		StartTime: varz.Start,
		IsLocal:   true,
	}

	// Get cluster port if clustering is enabled
	if varz.Cluster.Port > 0 {
		node.ClusterPort = varz.Cluster.Port
	}

	// Get RAM from system
	if vmStat, err := mem.VirtualMemory(); err == nil {
		node.RAMTotal = vmStat.Total
	}

	// GPU detection would require additional libraries
	// For now, we'll leave GPU fields at zero
	// In a production implementation, you'd use something like nvidia-smi
	// or a GPU detection library

	return node, nil
}

// getPeerNodes returns information about peer nodes in the cluster.
func (r *Reader) getPeerNodes() ([]NodeInfo, error) {
	var peers []NodeInfo

	// Get route information using the server's monitoring capabilities
	// The server provides Routez() for route information
	routez, err := r.server.Routez(&server.RoutezOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}

	for _, route := range routez.Routes {
		peer := NodeInfo{
			ID:         route.RemoteID,
			Name:       route.RemoteID, // May not have name, use ID
			ClusterURL: route.IP,
			IsLocal:    false,
		}

		// Route info doesn't include RAM/CPU of remote nodes
		// These would need to be obtained through a custom protocol
		// or by querying the remote node's monitoring endpoint

		peers = append(peers, peer)
	}

	return peers, nil
}

// GetSubscriptions returns all subscriptions for the local server.
// This is used for process detection.
func (r *Reader) GetSubscriptions() ([]SubscriptionInfo, error) {
	var subs []SubscriptionInfo

	// Get subscription information using Subsz
	subsz, err := r.server.Subsz(&server.SubszOptions{
		Subscriptions: true,
		Limit:         1000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	if subsz.Subs != nil {
		for _, sub := range subsz.Subs {
			subs = append(subs, SubscriptionInfo{
				Subject:  sub.Subject,
				Queue:    sub.Queue,
				NumMsgs:  uint64(sub.Msgs),
				NumBytes: 0, // Bytes not available in SubDetail
			})
		}
	}

	return subs, nil
}

// GetJetStreamInfo returns information about JetStream streams and consumers.
func (r *Reader) GetJetStreamInfo() ([]StreamInfo, []ConsumerInfo, error) {
	var streams []StreamInfo
	var consumers []ConsumerInfo

	// Get JetStream information using Jsz
	jsz, err := r.server.Jsz(&server.JSzOptions{
		Streams:   true,
		Consumer:  true,
		Config:    true,
		LeaderOnly: false,
	})
	if err != nil {
		// JetStream might not be enabled
		return streams, consumers, nil
	}

	// Process account info
	if jsz.AccountDetails != nil {
		for _, acc := range jsz.AccountDetails {
			for _, stream := range acc.Streams {
				si := StreamInfo{
					Name:      stream.Name,
					Subjects:  stream.Config.Subjects,
					Messages:  stream.State.Msgs,
					Bytes:     stream.State.Bytes,
					Consumers: stream.State.Consumers,
				}
				streams = append(streams, si)

				// Get consumers for this stream
				for _, consumer := range stream.Consumer {
					ci := ConsumerInfo{
						StreamName:    stream.Name,
						Name:          consumer.Name,
						FilterSubject: consumer.Config.FilterSubject,
						NumPending:    consumer.NumPending,
						NumAckPending: uint64(consumer.NumAckPending),
					}
					consumers = append(consumers, ci)
				}
			}
		}
	}

	return streams, consumers, nil
}

// GetVarz returns general server statistics.
func (r *Reader) GetVarz() (*server.Varz, error) {
	return r.server.Varz(&server.VarzOptions{})
}

// GetConnz returns connection information.
func (r *Reader) GetConnz() (*server.Connz, error) {
	return r.server.Connz(&server.ConnzOptions{
		Subscriptions: true,
		Limit:         1000,
	})
}
