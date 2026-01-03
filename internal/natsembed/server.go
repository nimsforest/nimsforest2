// Package natsembed provides an embedded NATS server for NimsForest.
package natsembed

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// Server wraps an embedded NATS server with JetStream support.
type Server struct {
	ns     *server.Server
	config Config
}

// Config holds configuration for the embedded NATS server.
type Config struct {
	NodeName    string   // hostname/node identifier
	ClusterName string   // cluster/forest ID for clustering
	DataDir     string   // directory for JetStream storage (e.g., /var/lib/nimsforest/jetstream)
	Peers       []string // cluster peer addresses (e.g., ["[2a01::1]:6222", "[2a01::2]:6222"])
	ClientPort  int      // client connection port (default: 4222)
	ClusterPort int      // cluster port (default: 6222)
	MonitorPort int      // HTTP monitoring port (default: 8222, -1 to disable)
}

// New creates a new embedded NATS server with the given configuration.
// NodeName and ClusterName are required.
func New(cfg Config) (*Server, error) {
	// Validate required fields
	if cfg.NodeName == "" {
		return nil, fmt.Errorf("NodeName is required")
	}
	if cfg.ClusterName == "" {
		return nil, fmt.Errorf("ClusterName is required")
	}

	// Apply defaults for ports
	// ClientPort of 0 means use a random available port (useful for tests)
	// ClientPort of -1 or unset defaults to the standard NATS port (4222)
	clientPort := cfg.ClientPort
	if clientPort == 0 {
		clientPort = server.RANDOM_PORT // -1, tells NATS to pick a free port
	} else if clientPort < 0 {
		clientPort = 4222 // Default NATS port
	}

	// ClusterPort defaults
	if cfg.ClusterPort == 0 {
		cfg.ClusterPort = 6222
	}
	// MonitorPort defaults to 8222 for HTTP monitoring; use -1 to explicitly disable
	if cfg.MonitorPort == 0 {
		cfg.MonitorPort = 8222
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "/var/lib/nimsforest/jetstream"
	}

	// Build NATS server options
	opts := &server.Options{
		ServerName: cfg.NodeName,
		Port:       clientPort,
		JetStream:  true,
		StoreDir:   cfg.DataDir,
	}

	// Configure HTTP monitoring if enabled
	if cfg.MonitorPort > 0 {
		opts.HTTPPort = cfg.MonitorPort
	}

	// Configure clustering only if there are peers
	// NATS JetStream requires actual routes when clustering is enabled
	if len(cfg.Peers) > 0 {
		// Get local IPv6 for cluster advertising
		localIP := getLocalIPv6()

		opts.Cluster = server.ClusterOpts{
			Name: cfg.ClusterName,
			Port: cfg.ClusterPort,
		}

		// Advertise our cluster address (IPv6)
		if localIP != "" {
			opts.Cluster.Advertise = fmt.Sprintf("[%s]:%d", localIP, cfg.ClusterPort)
		}

		// Build routes to peers as URLs
		routes := make([]*url.URL, 0, len(cfg.Peers))
		for _, peer := range cfg.Peers {
			routeURL, err := url.Parse(fmt.Sprintf("nats://%s", peer))
			if err != nil {
				log.Printf("[NATSEmbed] Warning: invalid peer address %s: %v", peer, err)
				continue
			}
			routes = append(routes, routeURL)
		}
		opts.Routes = routes

		log.Printf("[NATSEmbed] Clustering enabled: name=%s, port=%d, peers=%v",
			cfg.ClusterName, cfg.ClusterPort, cfg.Peers)
	} else {
		log.Printf("[NATSEmbed] Starting as first node in cluster: %s", cfg.ClusterName)
	}

	// Create the server
	ns, err := server.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS server: %w", err)
	}

	// Configure logging
	ns.ConfigureLogger()

	return &Server{
		ns:     ns,
		config: cfg,
	}, nil
}

// Start starts the embedded NATS server.
func (s *Server) Start() error {
	// Start the server in the background
	go s.ns.Start()

	// Wait for the server to be ready
	if !s.ns.ReadyForConnections(10 * time.Second) {
		return fmt.Errorf("NATS server failed to start within timeout")
	}

	monitorInfo := "disabled"
	if s.config.MonitorPort > 0 {
		monitorInfo = fmt.Sprintf("http://127.0.0.1:%d", s.config.MonitorPort)
	}
	log.Printf("[NATSEmbed] Server started: client=%s:%d, jetstream=%s, monitor=%s",
		s.ns.ID(), s.config.ClientPort, s.config.DataDir, monitorInfo)

	return nil
}

// ClientConn returns a NATS client connection to the embedded server.
func (s *Server) ClientConn() (*nats.Conn, error) {
	// Connect using the in-process URL for efficiency
	clientURL := s.ns.ClientURL()

	nc, err := nats.Connect(clientURL,
		nats.Name("nimsforest-internal"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("[NATSEmbed] Client disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[NATSEmbed] Client reconnected to %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to embedded NATS: %w", err)
	}

	return nc, nil
}

// JetStream returns a JetStream context from a client connection to the embedded server.
func (s *Server) JetStream() (nats.JetStreamContext, error) {
	nc, err := s.ClientConn()
	if err != nil {
		return nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return js, nil
}

// Shutdown gracefully shuts down the embedded NATS server.
func (s *Server) Shutdown() {
	if s.ns != nil {
		log.Printf("[NATSEmbed] Shutting down server...")
		s.ns.Shutdown()
		s.ns.WaitForShutdown()
		log.Printf("[NATSEmbed] Server shutdown complete")
	}
}

// WaitForShutdown blocks until the server has shut down.
func (s *Server) WaitForShutdown() {
	if s.ns != nil {
		s.ns.WaitForShutdown()
	}
}

// IsRunning returns true if the server is currently running.
func (s *Server) IsRunning() bool {
	return s.ns != nil && s.ns.Running()
}

// NumRoutes returns the number of active routes/peers in the cluster.
func (s *Server) NumRoutes() int {
	if s.ns == nil {
		return 0
	}
	return s.ns.NumRoutes()
}

// ClientURL returns the URL for client connections.
func (s *Server) ClientURL() string {
	if s.ns == nil {
		return ""
	}
	return s.ns.ClientURL()
}

// MonitorURL returns the URL for HTTP monitoring, or empty string if disabled.
func (s *Server) MonitorURL() string {
	if s.ns == nil || s.config.MonitorPort <= 0 {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", s.config.MonitorPort)
}

// getLocalIPv6 returns the non-loopback local IPv6 address of the host.
func getLocalIPv6() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// Only return IPv6 addresses (not IPv4-mapped)
			if ipnet.IP.To16() != nil && ipnet.IP.To4() == nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

// GetLocalIPv6 is exported for use by other packages.
func GetLocalIPv6() string {
	return getLocalIPv6()
}
