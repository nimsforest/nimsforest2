package httputil

import (
	"crypto/tls"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// CreateHTTPClient creates an HTTP client with proper timeout and TLS configuration
func CreateHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// IsRestrictedEnvironment checks if we're running in a restricted environment
// like Termux on Android where some operations may fail
func IsRestrictedEnvironment() bool {
	// Check for Termux-specific environment variable
	if os.Getenv("TERMUX_VERSION") != "" {
		return true
	}
	// Check for Termux prefix path
	if os.Getenv("PREFIX") == "/data/data/com.termux/files/usr" {
		return true
	}
	// Check if home directory is in Termux path
	home := os.Getenv("HOME")
	if home != "" && (home == "/data/data/com.termux/files/home" ||
		filepath.HasPrefix(home, "/data/data/com.termux")) {
		return true
	}
	return false
}

// GetPlatform returns the current platform string (e.g., "linux/amd64")
func GetPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// GetBinaryName returns the appropriate binary name for the current platform
func GetBinaryName(baseName string) string {
	return baseName + "-" + runtime.GOOS + "-" + runtime.GOARCH
}
