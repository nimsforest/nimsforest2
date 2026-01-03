package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Archive provides SQLite-based storage for inactive/historical data.
// It should be stored on a network drive for offline accessibility.
type Archive struct {
	db   *sql.DB
	path string
}

// ArchivedEntity represents an entity stored in the archive.
type ArchivedEntity struct {
	Key        string          `json:"key"`
	Data       json.RawMessage `json:"data"`
	ArchivedAt time.Time       `json:"archived_at"`
	ArchivedBy string          `json:"archived_by"` // Node that archived it
}

// NewArchive opens or creates an archive at the given path.
// The path should be on a network drive for offline accessibility.
func NewArchive(path string) (*Archive, error) {
	if path == "" {
		return nil, fmt.Errorf("archive path is required")
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive database: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS entities (
			key TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			archived_at DATETIME NOT NULL,
			archived_by TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create archive table: %w", err)
	}

	log.Printf("[Archive] Opened archive at %s", path)
	return &Archive{db: db, path: path}, nil
}

// Store saves an entity to the archive.
func (a *Archive) Store(key string, data []byte, archivedBy string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	_, err := a.db.Exec(
		`INSERT OR REPLACE INTO entities (key, data, archived_at, archived_by) VALUES (?, ?, ?, ?)`,
		key, string(data), time.Now().UTC(), archivedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to store entity %s: %w", key, err)
	}

	log.Printf("[Archive] Stored entity: %s", key)
	return nil
}

// Get retrieves an entity from the archive.
func (a *Archive) Get(key string) (*ArchivedEntity, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}

	var entity ArchivedEntity
	var dataStr string
	err := a.db.QueryRow(
		`SELECT key, data, archived_at, archived_by FROM entities WHERE key = ?`,
		key,
	).Scan(&entity.Key, &dataStr, &entity.ArchivedAt, &entity.ArchivedBy)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("entity not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity %s: %w", key, err)
	}

	entity.Data = json.RawMessage(dataStr)
	return &entity, nil
}

// List returns all keys in the archive.
func (a *Archive) List() ([]string, error) {
	rows, err := a.db.Query(`SELECT key FROM entities ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// Delete removes an entity from the archive.
func (a *Archive) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	result, err := a.db.Exec(`DELETE FROM entities WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("failed to delete entity %s: %w", key, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found: %s", key)
	}

	log.Printf("[Archive] Deleted entity: %s", key)
	return nil
}

// Close closes the archive database.
func (a *Archive) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// ArchiveFromSoil moves an entity from Soil to Archive.
// If deleteFromSoil is true, the entity is removed from Soil after archiving.
func ArchiveFromSoil(soil *Soil, archive *Archive, key string, archivedBy string, deleteFromSoil bool) error {
	// Read from Soil
	data, _, err := soil.Dig(key)
	if err != nil {
		return fmt.Errorf("failed to read from soil: %w", err)
	}

	// Store in Archive
	if err := archive.Store(key, data, archivedBy); err != nil {
		return fmt.Errorf("failed to store in archive: %w", err)
	}

	// Optionally delete from Soil
	if deleteFromSoil {
		if err := soil.Delete(key); err != nil {
			log.Printf("[Archive] Warning: archived but failed to delete from soil: %v", err)
		}
	}

	log.Printf("[Archive] Archived entity %s from soil (deleted=%v)", key, deleteFromSoil)
	return nil
}
