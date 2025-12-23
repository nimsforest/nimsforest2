package core

import (
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// Soil represents a JetStream Key-Value store for current state.
// It provides optimistic locking for concurrent updates.
type Soil struct {
	kv nats.KeyValue
}

// NewSoil creates a new Soil backed by a JetStream KV bucket.
// The bucket is created if it doesn't exist, with the name "SOIL".
func NewSoil(js nats.JetStreamContext) (*Soil, error) {
	bucketName := "SOIL"

	// Try to get existing bucket
	kv, err := js.KeyValue(bucketName)
	if err != nil {
		// Bucket doesn't exist, create it
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket:      bucketName,
			Description: "NimsForest current state storage",
			History:     10,                    // Keep last 10 revisions
			TTL:         0,                     // No TTL - data persists
			Storage:     nats.FileStorage,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create KV bucket %s: %w", bucketName, err)
		}
		log.Printf("[Soil] Created KV bucket: %s", bucketName)
	} else {
		log.Printf("[Soil] Using existing KV bucket: %s", bucketName)
	}

	return &Soil{
		kv: kv,
	}, nil
}

// Dig reads the current state of an entity from soil.
// Returns the data, the current revision number, and any error.
// The revision number is needed for optimistic locking with Bury.
func (s *Soil) Dig(entity string) ([]byte, uint64, error) {
	if entity == "" {
		return nil, 0, fmt.Errorf("entity key cannot be empty")
	}

	entry, err := s.kv.Get(entity)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return nil, 0, fmt.Errorf("entity not found: %s", entity)
		}
		return nil, 0, fmt.Errorf("failed to dig entity %s: %w", entity, err)
	}

	log.Printf("[Soil] Dug entity: %s (revision: %d, size: %d bytes)", 
		entity, entry.Revision(), len(entry.Value()))
	return entry.Value(), entry.Revision(), nil
}

// Bury writes state to soil with optimistic locking.
// If expectedRevision is 0, this is a new entity (create).
// If expectedRevision > 0, it must match the current revision (update).
// Returns an error if the revision doesn't match (concurrent modification).
func (s *Soil) Bury(entity string, data []byte, expectedRevision uint64) error {
	if entity == "" {
		return fmt.Errorf("entity key cannot be empty")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	var err error
	
	if expectedRevision == 0 {
		// Create - entity should not exist
		_, err = s.kv.Create(entity, data)
		if err != nil {
			// Check if it's because key already exists
			if err == nats.ErrKeyExists {
				return fmt.Errorf("entity already exists: %s (use non-zero revision for updates)", entity)
			}
			return fmt.Errorf("failed to create entity %s: %w", entity, err)
		}
		log.Printf("[Soil] Buried new entity: %s (size: %d bytes)", entity, len(data))
	} else {
		// Update - check revision matches
		_, err = s.kv.Update(entity, data, expectedRevision)
		if err != nil {
			// Check for revision mismatch
			if err == nats.ErrKeyExists {
				return fmt.Errorf("entity %s was modified (revision mismatch): %w", entity, err)
			}
			return fmt.Errorf("failed to update entity %s: %w", entity, err)
		}
		log.Printf("[Soil] Buried update to entity: %s (revision: %d, size: %d bytes)", 
			entity, expectedRevision, len(data))
	}

	return nil
}

// Put writes state to soil without checking the revision.
// This is less safe than Bury but useful when you don't care about concurrent updates.
func (s *Soil) Put(entity string, data []byte) (uint64, error) {
	if entity == "" {
		return 0, fmt.Errorf("entity key cannot be empty")
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("data cannot be empty")
	}

	revision, err := s.kv.Put(entity, data)
	if err != nil {
		return 0, fmt.Errorf("failed to put entity %s: %w", entity, err)
	}

	log.Printf("[Soil] Put entity: %s (revision: %d)", entity, revision)
	return revision, nil
}

// Delete removes an entity from soil.
// Note: NATS KV delete is idempotent and doesn't return an error if the key doesn't exist.
func (s *Soil) Delete(entity string) error {
	if entity == "" {
		return fmt.Errorf("entity key cannot be empty")
	}

	// Check if entity exists first
	_, err := s.kv.Get(entity)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return fmt.Errorf("entity not found: %s", entity)
		}
		return fmt.Errorf("failed to check entity %s: %w", entity, err)
	}

	err = s.kv.Delete(entity)
	if err != nil {
		return fmt.Errorf("failed to delete entity %s: %w", entity, err)
	}

	log.Printf("[Soil] Deleted entity: %s", entity)
	return nil
}

// Watch observes changes to entities matching a pattern.
// The pattern can include wildcards (> for multi-level).
// The handler is called for each change.
func (s *Soil) Watch(pattern string, handler func(entity string, data []byte, revision uint64)) error {
	if pattern == "" {
		pattern = ">"
	}

	// Create a watcher
	watcher, err := s.kv.Watch(pattern)
	if err != nil {
		return fmt.Errorf("failed to watch pattern %s: %w", pattern, err)
	}

	// Start watching in a goroutine
	go func() {
		for entry := range watcher.Updates() {
			if entry == nil {
				continue
			}
			
			// Skip deleted entries
			if entry.Operation() == nats.KeyValueDelete || entry.Operation() == nats.KeyValuePurge {
				log.Printf("[Soil] Watched entity deleted: %s", entry.Key())
				continue
			}

			handler(entry.Key(), entry.Value(), entry.Revision())
		}
	}()

	log.Printf("[Soil] Watching pattern: %s", pattern)
	return nil
}

// WatchAll watches all changes to the bucket.
func (s *Soil) WatchAll(handler func(entity string, data []byte, revision uint64)) error {
	return s.Watch(">", handler)
}

// Keys returns all keys in the bucket.
func (s *Soil) Keys() ([]string, error) {
	keys, err := s.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}
	return keys, nil
}

// Status returns the status of the KV bucket.
func (s *Soil) Status() (nats.KeyValueStatus, error) {
	status, err := s.kv.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket status: %w", err)
	}
	return status, nil
}
