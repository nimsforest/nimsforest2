package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// UnixBedrock provides direct filesystem access as a bedrock.
// It uses fsnotify for watching file changes on Linux, macOS, and BSD.
type UnixBedrock struct {
	name     string
	root     string
	readonly bool

	watcher *fsnotify.Watcher
	handler func(BedrockEvent)

	mu       sync.RWMutex
	running  bool
	cancelFn context.CancelFunc
}

// UnixBedrockConfig configures a Unix bedrock.
type UnixBedrockConfig struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path"`
	ReadOnly bool   `yaml:"readonly,omitempty"`
}

// NewUnixBedrock creates a new Unix bedrock.
func NewUnixBedrock(cfg UnixBedrockConfig) (*UnixBedrock, error) {
	// Validate and resolve path
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create it
			if err := os.MkdirAll(absPath, 0755); err != nil {
				return nil, fmt.Errorf("path does not exist and could not be created: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to stat path: %w", err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	return &UnixBedrock{
		name:     cfg.Name,
		root:     absPath,
		readonly: cfg.ReadOnly,
	}, nil
}

// Name returns the bedrock name.
func (b *UnixBedrock) Name() string {
	return b.name
}

// Type returns "unix".
func (b *UnixBedrock) Type() string {
	return "unix"
}

// Root returns the root path.
func (b *UnixBedrock) Root() string {
	return b.root
}

// Start starts the bedrock and begins watching for changes.
func (b *UnixBedrock) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return fmt.Errorf("bedrock already running")
	}

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	b.watcher = watcher

	// Add all directories recursively
	err = filepath.WalkDir(b.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if d.IsDir() {
			if err := watcher.Add(path); err != nil {
				log.Printf("[Bedrock:%s] Warning: failed to watch %s: %v", b.name, path, err)
			}
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	b.cancelFn = cancel

	// Start watching in background
	go b.watchLoop(ctx)

	b.running = true
	log.Printf("[Bedrock:%s] Started watching %s", b.name, b.root)

	// Emit mounted event
	if b.handler != nil {
		b.handler(BedrockEvent{
			Bedrock:   b.name,
			Type:      BedrockEventMounted,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// Stop stops the bedrock.
func (b *UnixBedrock) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	if b.cancelFn != nil {
		b.cancelFn()
	}

	if b.watcher != nil {
		b.watcher.Close()
		b.watcher = nil
	}

	b.running = false
	log.Printf("[Bedrock:%s] Stopped", b.name)

	// Emit unmounted event
	if b.handler != nil {
		b.handler(BedrockEvent{
			Bedrock:   b.name,
			Type:      BedrockEventUnmounted,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// watchLoop handles file system events.
func (b *UnixBedrock) watchLoop(ctx context.Context) {
	for {
		b.mu.RLock()
		watcher := b.watcher
		b.mu.RUnlock()

		if watcher == nil {
			return
		}

		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			b.handleFSEvent(event)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[Bedrock:%s] Watch error: %v", b.name, err)
		}
	}
}

// handleFSEvent converts fsnotify events to bedrock events.
func (b *UnixBedrock) handleFSEvent(event fsnotify.Event) {
	if b.handler == nil {
		return
	}

	// Get relative path
	relPath, err := filepath.Rel(b.root, event.Name)
	if err != nil {
		relPath = event.Name
	}

	// Skip hidden files and common ignore patterns
	if shouldIgnore(relPath) {
		return
	}

	var eventType BedrockEventType
	var fileInfo *FileInfo

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = BedrockEventFileCreated
		fileInfo, _ = b.stat(relPath)

		// If it's a new directory, add it to the watcher
		if fileInfo != nil && fileInfo.IsDir {
			b.mu.RLock()
			if b.watcher != nil {
				_ = b.watcher.Add(event.Name)
			}
			b.mu.RUnlock()
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = BedrockEventFileModified
		fileInfo, _ = b.stat(relPath)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = BedrockEventFileDeleted

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = BedrockEventFileDeleted // Rename is essentially delete + create

	default:
		return
	}

	b.handler(BedrockEvent{
		Bedrock:   b.name,
		Type:      eventType,
		Path:      relPath,
		Timestamp: time.Now(),
		FileInfo:  fileInfo,
	})
}

// List returns files in the given path.
func (b *UnixBedrock) List(path string) ([]FileInfo, error) {
	b.mu.RLock()
	running := b.running
	b.mu.RUnlock()

	if !running {
		return nil, ErrBedrockStopped
	}

	fullPath := filepath.Join(b.root, path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		entryPath := filepath.Join(path, entry.Name())
		files = append(files, FileInfo{
			Path:     entryPath,
			Name:     entry.Name(),
			Size:     info.Size(),
			Modified: info.ModTime(),
			IsDir:    entry.IsDir(),
			MimeType: getMimeType(entry.Name()),
		})
	}

	return files, nil
}

// Read reads a file's contents.
func (b *UnixBedrock) Read(path string) ([]byte, error) {
	b.mu.RLock()
	running := b.running
	b.mu.RUnlock()

	if !running {
		return nil, ErrBedrockStopped
	}

	// Validate path
	if err := validatePath(path); err != nil {
		return nil, err
	}

	fullPath := filepath.Join(b.root, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Stat returns file metadata.
func (b *UnixBedrock) Stat(path string) (*FileInfo, error) {
	b.mu.RLock()
	running := b.running
	b.mu.RUnlock()

	if !running {
		return nil, ErrBedrockStopped
	}

	return b.stat(path)
}

// stat is the internal stat implementation.
func (b *UnixBedrock) stat(path string) (*FileInfo, error) {
	fullPath := filepath.Join(b.root, path)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileInfo := &FileInfo{
		Path:     path,
		Name:     info.Name(),
		Size:     info.Size(),
		Modified: info.ModTime(),
		IsDir:    info.IsDir(),
		MimeType: getMimeType(info.Name()),
	}

	// Compute content hash for files (not directories)
	if !info.IsDir() && info.Size() < 10*1024*1024 { // Only for files < 10MB
		if data, err := os.ReadFile(fullPath); err == nil {
			hash := sha256.Sum256(data)
			fileInfo.ContentHash = "sha256:" + hex.EncodeToString(hash[:])
		}
	}

	return fileInfo, nil
}

// Tree returns a text representation of the directory tree.
func (b *UnixBedrock) Tree(maxDepth int) (string, error) {
	b.mu.RLock()
	running := b.running
	b.mu.RUnlock()

	if !running {
		return "", ErrBedrockStopped
	}

	if maxDepth <= 0 {
		maxDepth = 10 // Default max depth
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%s/ (mounted: %s, type: %s)\n", b.name, b.root, b.Type()))

	var fileCount, dirCount int
	var totalSize int64

	err := b.buildTree(&buf, "", 0, maxDepth, &fileCount, &dirCount, &totalSize)
	if err != nil {
		return "", err
	}

	buf.WriteString(fmt.Sprintf("\n%d files, %d directories, %s total\n",
		fileCount, dirCount, formatSize(totalSize)))

	return buf.String(), nil
}

// buildTree recursively builds the tree representation.
func (b *UnixBedrock) buildTree(buf *bytes.Buffer, path string, depth, maxDepth int, fileCount, dirCount *int, totalSize *int64) error {
	if depth >= maxDepth {
		return nil
	}

	fullPath := filepath.Join(b.root, path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entryPath := filepath.Join(path, entry.Name())
		isLast := i == len(entries)-1

		// Build prefix
		prefix := strings.Repeat("│   ", depth)
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			*dirCount++
			buf.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, entry.Name()))
			b.buildTree(buf, entryPath, depth+1, maxDepth, fileCount, dirCount, totalSize)
		} else {
			*fileCount++
			*totalSize += info.Size()
			buf.WriteString(fmt.Sprintf("%s%s%s (%s, modified: %s)\n",
				prefix, connector, entry.Name(),
				formatSize(info.Size()),
				info.ModTime().Format("2006-01-02")))
		}
	}

	return nil
}

// Write writes content to a file.
func (b *UnixBedrock) Write(path string, content []byte) error {
	b.mu.RLock()
	running := b.running
	readonly := b.readonly
	b.mu.RUnlock()

	if !running {
		return ErrBedrockStopped
	}
	if readonly {
		return ErrReadOnly
	}

	// Validate path
	if err := validatePath(path); err != nil {
		return err
	}

	fullPath := filepath.Join(b.root, path)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("[Bedrock:%s] Wrote %s (%d bytes)", b.name, path, len(content))
	return nil
}

// Delete removes a file or directory.
func (b *UnixBedrock) Delete(path string) error {
	b.mu.RLock()
	running := b.running
	readonly := b.readonly
	b.mu.RUnlock()

	if !running {
		return ErrBedrockStopped
	}
	if readonly {
		return ErrReadOnly
	}

	// Validate path
	if err := validatePath(path); err != nil {
		return err
	}

	fullPath := filepath.Join(b.root, path)

	// Check if exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to stat: %w", err)
	}

	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	log.Printf("[Bedrock:%s] Deleted %s", b.name, path)
	return nil
}

// Move moves a file or directory.
func (b *UnixBedrock) Move(from, to string) error {
	b.mu.RLock()
	running := b.running
	readonly := b.readonly
	b.mu.RUnlock()

	if !running {
		return ErrBedrockStopped
	}
	if readonly {
		return ErrReadOnly
	}

	// Validate paths
	if err := validatePath(from); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := validatePath(to); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	fromPath := filepath.Join(b.root, from)
	toPath := filepath.Join(b.root, to)

	// Check source exists
	if _, err := os.Stat(fromPath); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Ensure destination parent exists
	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := os.Rename(fromPath, toPath); err != nil {
		return fmt.Errorf("failed to move: %w", err)
	}

	log.Printf("[Bedrock:%s] Moved %s -> %s", b.name, from, to)
	return nil
}

// IsReadOnly returns whether the bedrock is read-only.
func (b *UnixBedrock) IsReadOnly() bool {
	return b.readonly
}

// SupportsWatch returns true as Unix bedrocks support fsnotify.
func (b *UnixBedrock) SupportsWatch() bool {
	return true
}

// Watch sets the event handler for file changes.
func (b *UnixBedrock) Watch(ctx context.Context, handler func(BedrockEvent)) error {
	b.mu.Lock()
	b.handler = handler
	b.mu.Unlock()
	return nil
}

// Manifest returns the bedrock manifest.
func (b *UnixBedrock) Manifest() (*BedrockManifest, error) {
	b.mu.RLock()
	running := b.running
	b.mu.RUnlock()

	if !running {
		return nil, ErrBedrockStopped
	}

	var fileCount int
	var totalSize int64

	err := filepath.WalkDir(b.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			fileCount++
			if info, err := d.Info(); err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return &BedrockManifest{
		Name:      b.name,
		Type:      b.Type(),
		Root:      b.root,
		FileCount: fileCount,
		TotalSize: totalSize,
		LastScan:  time.Now(),
	}, nil
}

// Helper functions

// validatePath ensures the path is safe (no path traversal).
func validatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	// Clean and check for path traversal
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "..") || strings.HasPrefix(clean, "/") {
		return ErrInvalidPath
	}

	return nil
}

// shouldIgnore returns true if the path should be ignored.
func shouldIgnore(path string) bool {
	name := filepath.Base(path)

	// Hidden files
	if strings.HasPrefix(name, ".") {
		return true
	}

	// Common ignore patterns
	ignorePatterns := []string{
		"node_modules",
		"__pycache__",
		".git",
		".svn",
		".hg",
		"vendor",
		"target",
		".idea",
		".vscode",
	}

	for _, pattern := range ignorePatterns {
		if name == pattern || strings.Contains(path, "/"+pattern+"/") {
			return true
		}
	}

	return false
}

// getMimeType returns the MIME type for a file based on extension.
func getMimeType(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return "application/octet-stream"
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Common types not in stdlib
		switch ext {
		case ".md":
			return "text/markdown"
		case ".yaml", ".yml":
			return "text/yaml"
		case ".go":
			return "text/x-go"
		case ".lua":
			return "text/x-lua"
		case ".ts":
			return "text/typescript"
		case ".tsx":
			return "text/typescript-jsx"
		case ".jsx":
			return "text/javascript-jsx"
		default:
			return "application/octet-stream"
		}
	}
	return mimeType
}

// formatSize formats bytes as human-readable string.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
