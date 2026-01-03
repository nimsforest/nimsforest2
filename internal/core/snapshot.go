package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Snapshot represents a point-in-time export of all state from Soil.
// It can be stored on a network drive for offline review and recovery.
type Snapshot struct {
	Version   string                 `json:"version"`    // Snapshot format version
	Timestamp time.Time              `json:"timestamp"`  // When the snapshot was taken
	NodeName  string                 `json:"node_name"`  // Node that created the snapshot
	EntityCount int                  `json:"entity_count"` // Number of entities in snapshot
	Entities  map[string]EntityState `json:"entities"`   // All entity states
}

// EntityState represents the state of a single entity in a snapshot.
type EntityState struct {
	Key      string          `json:"key"`
	Data     json.RawMessage `json:"data"`
	Revision uint64          `json:"revision"`
}

// SnapshotConfig holds configuration for the snapshot manager.
type SnapshotConfig struct {
	// Directory where snapshots are stored (should be a network drive for durability)
	Dir string

	// How often to take snapshots (e.g., 5 * time.Minute)
	Interval time.Duration

	// Node name for identifying which node created the snapshot
	NodeName string

	// Maximum number of snapshots to retain (0 = unlimited)
	MaxSnapshots int
}

// SnapshotManager handles periodic snapshots of Soil state.
type SnapshotManager struct {
	soil   *Soil
	config SnapshotConfig
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager(soil *Soil, config SnapshotConfig) (*SnapshotManager, error) {
	if config.Dir == "" {
		return nil, fmt.Errorf("snapshot directory is required")
	}
	if config.Interval <= 0 {
		config.Interval = 5 * time.Minute // Default: every 5 minutes
	}
	if config.NodeName == "" {
		hostname, _ := os.Hostname()
		config.NodeName = hostname
	}
	if config.MaxSnapshots == 0 {
		config.MaxSnapshots = 10 // Default: keep last 10 snapshots
	}

	// Ensure snapshot directory exists
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory %s: %w", config.Dir, err)
	}

	return &SnapshotManager{
		soil:   soil,
		config: config,
	}, nil
}

// Start begins periodic snapshotting in the background.
func (m *SnapshotManager) Start() {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	go func() {
		// Take an initial snapshot
		if err := m.TakeSnapshot(); err != nil {
			log.Printf("[Snapshot] Initial snapshot failed: %v", err)
		}

		ticker := time.NewTicker(m.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				log.Printf("[Snapshot] Stopping periodic snapshots")
				return
			case <-ticker.C:
				if err := m.TakeSnapshot(); err != nil {
					log.Printf("[Snapshot] Periodic snapshot failed: %v", err)
				}
			}
		}
	}()

	log.Printf("[Snapshot] Started periodic snapshots every %v to %s", m.config.Interval, m.config.Dir)
}

// Stop stops the periodic snapshotting.
func (m *SnapshotManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// TakeSnapshot creates a snapshot of the current Soil state.
func (m *SnapshotManager) TakeSnapshot() error {
	startTime := time.Now()

	// Get all keys from Soil
	keys, err := m.soil.Keys()
	if err != nil {
		// No keys is not an error - just an empty snapshot
		if err.Error() == "failed to get keys: nats: no keys found" {
			keys = []string{}
		} else {
			return fmt.Errorf("failed to get keys: %w", err)
		}
	}

	// Build the snapshot
	snapshot := Snapshot{
		Version:     "1.0",
		Timestamp:   startTime,
		NodeName:    m.config.NodeName,
		EntityCount: len(keys),
		Entities:    make(map[string]EntityState, len(keys)),
	}

	// Read each entity
	for _, key := range keys {
		data, revision, err := m.soil.Dig(key)
		if err != nil {
			log.Printf("[Snapshot] Warning: failed to read entity %s: %v", key, err)
			continue
		}

		snapshot.Entities[key] = EntityState{
			Key:      key,
			Data:     json.RawMessage(data),
			Revision: revision,
		}
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("snapshot_%s.json", startTime.Format("20060102_150405"))
	filepath := filepath.Join(m.config.Dir, filename)

	// Write to file
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot to %s: %w", filepath, err)
	}

	duration := time.Since(startTime)
	log.Printf("[Snapshot] Created %s (%d entities, %d bytes, took %v)",
		filename, snapshot.EntityCount, len(data), duration)

	// Cleanup old snapshots
	if err := m.cleanupOldSnapshots(); err != nil {
		log.Printf("[Snapshot] Warning: failed to cleanup old snapshots: %v", err)
	}

	return nil
}

// cleanupOldSnapshots removes old snapshots beyond MaxSnapshots.
func (m *SnapshotManager) cleanupOldSnapshots() error {
	if m.config.MaxSnapshots <= 0 {
		return nil // Unlimited retention
	}

	entries, err := os.ReadDir(m.config.Dir)
	if err != nil {
		return fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	// Filter to only snapshot files
	var snapshots []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			snapshots = append(snapshots, entry.Name())
		}
	}

	// Sort by name (which includes timestamp, so oldest first)
	sort.Strings(snapshots)

	// Remove oldest snapshots if we have too many
	toRemove := len(snapshots) - m.config.MaxSnapshots
	for i := 0; i < toRemove; i++ {
		path := filepath.Join(m.config.Dir, snapshots[i])
		if err := os.Remove(path); err != nil {
			log.Printf("[Snapshot] Warning: failed to remove old snapshot %s: %v", snapshots[i], err)
		} else {
			log.Printf("[Snapshot] Removed old snapshot: %s", snapshots[i])
		}
	}

	return nil
}

// ListSnapshots returns a list of available snapshots, newest first.
func (m *SnapshotManager) ListSnapshots() ([]string, error) {
	entries, err := os.ReadDir(m.config.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var snapshots []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			snapshots = append(snapshots, entry.Name())
		}
	}

	// Sort descending (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(snapshots)))

	return snapshots, nil
}

// LoadSnapshot loads a snapshot from a file.
func LoadSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file %s: %w", path, err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot file %s: %w", path, err)
	}

	return &snapshot, nil
}

// RestoreSnapshot restores state from a snapshot into Soil.
// This will overwrite existing entities with the same keys.
// Set clearExisting to true to delete all entities not in the snapshot.
func RestoreSnapshot(soil *Soil, snapshot *Snapshot, clearExisting bool) error {
	log.Printf("[Snapshot] Restoring snapshot from %s (%d entities, clearExisting=%v)",
		snapshot.Timestamp.Format(time.RFC3339), snapshot.EntityCount, clearExisting)

	// If clearing existing, get current keys and delete those not in snapshot
	if clearExisting {
		currentKeys, err := soil.Keys()
		if err == nil {
			snapshotKeys := make(map[string]bool, len(snapshot.Entities))
			for key := range snapshot.Entities {
				snapshotKeys[key] = true
			}

			for _, key := range currentKeys {
				if !snapshotKeys[key] {
					if err := soil.Delete(key); err != nil {
						log.Printf("[Snapshot] Warning: failed to delete entity %s during restore: %v", key, err)
					}
				}
			}
		}
	}

	// Restore each entity from snapshot
	var restored, failed int
	for key, entity := range snapshot.Entities {
		// Use Put to overwrite regardless of current state
		_, err := soil.Put(key, entity.Data)
		if err != nil {
			log.Printf("[Snapshot] Warning: failed to restore entity %s: %v", key, err)
			failed++
			continue
		}
		restored++
	}

	log.Printf("[Snapshot] Restore complete: %d restored, %d failed", restored, failed)

	if failed > 0 {
		return fmt.Errorf("restore completed with %d failures", failed)
	}
	return nil
}

// LatestSnapshotPath returns the path to the most recent snapshot file.
func (m *SnapshotManager) LatestSnapshotPath() (string, error) {
	snapshots, err := m.ListSnapshots()
	if err != nil {
		return "", err
	}
	if len(snapshots) == 0 {
		return "", fmt.Errorf("no snapshots found")
	}
	return filepath.Join(m.config.Dir, snapshots[0]), nil
}

// RestoreLatest restores from the most recent snapshot.
func (m *SnapshotManager) RestoreLatest(clearExisting bool) error {
	path, err := m.LatestSnapshotPath()
	if err != nil {
		return err
	}

	snapshot, err := LoadSnapshot(path)
	if err != nil {
		return err
	}

	return RestoreSnapshot(m.soil, snapshot, clearExisting)
}
