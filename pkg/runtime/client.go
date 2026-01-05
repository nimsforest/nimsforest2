// Package runtime provides the CLI client for NimsForest management API.
package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the CLI client for the NimsForest management API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = GetAPIURL()
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientFromEnv creates a client using the environment configuration.
func NewClientFromEnv() *Client {
	return NewClient(GetAPIURL())
}

// =============================================================================
// Health & Status
// =============================================================================

// Health checks if the API is healthy.
func (c *Client) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("cannot connect to nimsforest daemon at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}
	return nil
}

// Status returns the current forest status.
func (c *Client) Status() (*ForestStatus, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/status")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var status ForestStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &status, nil
}

// =============================================================================
// TreeHouse Operations
// =============================================================================

// ListTreeHouses returns all running treehouses.
func (c *Client) ListTreeHouses() ([]ComponentInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/treehouses")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var treehouses []ComponentInfo
	if err := json.NewDecoder(resp.Body).Decode(&treehouses); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return treehouses, nil
}

// AddTreeHouse adds a new treehouse.
func (c *Client) AddTreeHouse(name, subscribes, publishes, script string) error {
	payload := map[string]string{
		"name":       name,
		"subscribes": subscribes,
		"publishes":  publishes,
		"script":     script,
	}

	data, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/treehouses", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return c.parseError(resp)
	}
	return nil
}

// AddTreeHouseFromConfig adds a treehouse from a config struct.
func (c *Client) AddTreeHouseFromConfig(cfg TreeHouseConfig) error {
	return c.AddTreeHouse(cfg.Name, cfg.Subscribes, cfg.Publishes, cfg.Script)
}

// RemoveTreeHouse removes a treehouse by name.
func (c *Client) RemoveTreeHouse(name string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.baseURL+"/api/v1/treehouses/"+name, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}
	return nil
}

// =============================================================================
// Nim Operations
// =============================================================================

// ListNims returns all running nims.
func (c *Client) ListNims() ([]ComponentInfo, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/nims")
	if err != nil {
		return nil, fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var nims []ComponentInfo
	if err := json.NewDecoder(resp.Body).Decode(&nims); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return nims, nil
}

// AddNim adds a new nim.
func (c *Client) AddNim(name, subscribes, publishes, prompt string) error {
	payload := map[string]string{
		"name":       name,
		"subscribes": subscribes,
		"publishes":  publishes,
		"prompt":     prompt,
	}

	data, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(c.baseURL+"/api/v1/nims", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return c.parseError(resp)
	}
	return nil
}

// AddNimFromConfig adds a nim from a config struct.
func (c *Client) AddNimFromConfig(cfg NimConfig) error {
	return c.AddNim(cfg.Name, cfg.Subscribes, cfg.Publishes, cfg.Prompt)
}

// RemoveNim removes a nim by name.
func (c *Client) RemoveNim(name string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.baseURL+"/api/v1/nims/"+name, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}
	return nil
}

// =============================================================================
// Reload
// =============================================================================

// Reload reloads the forest configuration from disk.
func (c *Client) Reload() error {
	resp, err := c.httpClient.Post(c.baseURL+"/-/reload", "application/json", nil)
	if err != nil {
		return fmt.Errorf("cannot connect to nimsforest daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse as JSON error
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("%s", errResp.Error)
	}

	// Fall back to status text
	if len(body) > 0 {
		return fmt.Errorf("%s: %s", resp.Status, string(body))
	}
	return fmt.Errorf("%s", resp.Status)
}

// BaseURL returns the API base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}
