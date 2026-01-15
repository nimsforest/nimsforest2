package runtime

import (
	"context"
	"fmt"
	"time"
)

// Bedrock represents a persistent storage layer beneath Soil.
// While Soil (KV store) provides fast ephemeral access, Bedrock
// provides persistent storage that survives restarts.
//
// Bedrock is always the source of truth. Soil can be rebuilt from Bedrock.
type Bedrock interface {
	// Name returns the bedrock's name for identification
	Name() string

	// Type returns the bedrock type (git, unix, google_drive, s3)
	Type() string

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error

	// Read operations
	List(path string) ([]FileInfo, error)
	Read(path string) ([]byte, error)
	Stat(path string) (*FileInfo, error)
	Tree(maxDepth int) (string, error)

	// Write operations (may return ErrReadOnly)
	Write(path string, content []byte) error
	Delete(path string) error
	Move(from, to string) error

	// Capabilities
	IsReadOnly() bool
	SupportsWatch() bool
}

// FileInfo provides metadata about a file in a Bedrock.
type FileInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	Modified    time.Time `json:"modified"`
	IsDir       bool      `json:"is_dir"`
	ContentHash string    `json:"content_hash,omitempty"`
	MimeType    string    `json:"mime_type,omitempty"`
}

// BedrockManifest provides metadata about a mounted bedrock.
type BedrockManifest struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Root      string    `json:"root"`
	FileCount int       `json:"file_count"`
	TotalSize int64     `json:"total_size"`
	LastScan  time.Time `json:"last_scan"`

	// Git-specific fields
	Remote string `json:"remote,omitempty"`
	Branch string `json:"branch,omitempty"`
}

// BedrockLock represents a distributed lock on a file.
type BedrockLock struct {
	Holder   string     `json:"holder"`
	Type     LockType   `json:"type"`
	Acquired time.Time  `json:"acquired"`
	TTL      int        `json:"ttl,omitempty"` // seconds, 0 for pending_pr
	PR       string     `json:"pr,omitempty"`  // PR reference for pending_pr type
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// LockType indicates the type of lock held.
type LockType string

const (
	// LockTypeWrite is a short-lived lock for active writes.
	LockTypeWrite LockType = "write"

	// LockTypePendingPR is held until PR is merged/closed.
	LockTypePendingPR LockType = "pending_pr"
)

// IsExpired returns true if the lock has expired.
func (l *BedrockLock) IsExpired() bool {
	if l.Type == LockTypePendingPR {
		return false // PR locks don't expire
	}
	if l.ExpiresAt != nil {
		return time.Now().After(*l.ExpiresAt)
	}
	if l.TTL > 0 {
		return time.Since(l.Acquired) > time.Duration(l.TTL)*time.Second
	}
	return false
}

// BedrockEvent represents a change event from a bedrock.
type BedrockEvent struct {
	Bedrock   string         `json:"bedrock"`
	Type      BedrockEventType `json:"type"`
	Path      string         `json:"path,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	FileInfo  *FileInfo      `json:"file_info,omitempty"`
}

// BedrockEventType indicates the type of bedrock event.
type BedrockEventType string

const (
	BedrockEventMounted      BedrockEventType = "mounted"
	BedrockEventUnmounted    BedrockEventType = "unmounted"
	BedrockEventFileCreated  BedrockEventType = "file.created"
	BedrockEventFileModified BedrockEventType = "file.modified"
	BedrockEventFileDeleted  BedrockEventType = "file.deleted"
	BedrockEventFileMoved    BedrockEventType = "file.moved"
)

// PersistRequest represents a request to persist data to bedrock.
type PersistRequest struct {
	Bedrock string `json:"bedrock"`
	Path    string `json:"path"`
	Content []byte `json:"content,omitempty"`
	Message string `json:"message,omitempty"` // Commit message for git
}

// PersistResult represents the result of a persist operation.
type PersistResult struct {
	Bedrock string          `json:"bedrock"`
	Path    string          `json:"path"`
	Status  PersistStatus   `json:"status"`
	Error   string          `json:"error,omitempty"`
	PR      string          `json:"pr,omitempty"` // PR URL for pending status
}

// PersistStatus indicates the status of a persist operation.
type PersistStatus string

const (
	PersistStatusComplete PersistStatus = "complete"
	PersistStatusPending  PersistStatus = "pending"  // Awaiting approval
	PersistStatusRejected PersistStatus = "rejected" // PR was closed
	PersistStatusFailed   PersistStatus = "failed"
)

// Common errors
var (
	ErrReadOnly       = fmt.Errorf("bedrock is read-only")
	ErrNotFound       = fmt.Errorf("file not found")
	ErrAlreadyExists  = fmt.Errorf("file already exists")
	ErrLocked         = fmt.Errorf("file is locked")
	ErrAwaitingPR     = fmt.Errorf("file is locked pending PR approval")
	ErrInvalidPath    = fmt.Errorf("invalid path")
	ErrBedrockStopped = fmt.Errorf("bedrock is not running")
)

// LockedError provides details about a lock conflict.
type LockedError struct {
	Holder string
	Lock   *BedrockLock
}

func (e *LockedError) Error() string {
	if e.Lock != nil && e.Lock.Type == LockTypePendingPR {
		return fmt.Sprintf("file locked pending PR approval: %s (holder: %s)", e.Lock.PR, e.Holder)
	}
	return fmt.Sprintf("file is locked by %s", e.Holder)
}

// Soil key patterns for bedrock data
const (
	// SoilKeyBedrockTree is the pattern for tree documents
	// Format: bedrock:{name}:tree
	SoilKeyBedrockTree = "bedrock:%s:tree"

	// SoilKeyBedrockManifest is the pattern for manifests
	// Format: bedrock:{name}:manifest
	SoilKeyBedrockManifest = "bedrock:%s:manifest"

	// SoilKeyBedrockFile is the pattern for file metadata
	// Format: bedrock:{name}:file:{path}
	SoilKeyBedrockFile = "bedrock:%s:file:%s"

	// SoilKeyBedrockLock is the pattern for file locks
	// Format: bedrock:{name}:lock:{path}
	SoilKeyBedrockLock = "bedrock:%s:lock:%s"

	// SoilKeyBedrockCache is the pattern for cached file content
	// Format: cache:{name}:{path}
	SoilKeyBedrockCache = "cache:%s:%s"
)

// NATS subject patterns for bedrock events
const (
	// SubjectBedrockEvents is the pattern for bedrock events
	// Format: bedrock.{name}.{event_type}
	SubjectBedrockEvents = "bedrock.%s.%s"

	// SubjectPersistRequest is the pattern for persist requests
	// Format: persist.{name}.request
	SubjectPersistRequest = "persist.%s.request"

	// SubjectPersistResult is the pattern for persist results
	// Format: persist.{name}.{status}
	SubjectPersistResult = "persist.%s.%s"

	// SubjectIndexUpdated is emitted when bedrock index is updated
	// Format: index.{name}.updated
	SubjectIndexUpdated = "index.%s.updated"
)

// BedrockWatcher is an optional interface for bedrocks that support
// watching for file changes.
type BedrockWatcher interface {
	Bedrock

	// Watch starts watching for file changes and calls the handler
	// for each change event.
	Watch(ctx context.Context, handler func(BedrockEvent)) error
}

// GitBedrock is an optional interface for git-based bedrocks.
type GitBedrock interface {
	Bedrock

	// WriteMode returns "commit" or "pull_request"
	WriteMode() string

	// CreatePR creates a pull request for the given changes
	CreatePR(ctx context.Context, branch, title, body string) (prURL string, err error)

	// Sync syncs with the remote repository
	Sync(ctx context.Context) error

	// CurrentBranch returns the current branch name
	CurrentBranch() string

	// Remote returns the remote URL
	Remote() string
}
