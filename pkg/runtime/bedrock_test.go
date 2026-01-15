package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUnixBedrock_Basic(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "bedrock-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create bedrock
	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name: "test",
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	// Test basic properties
	if br.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", br.Name())
	}
	if br.Type() != "unix" {
		t.Errorf("Expected type 'unix', got '%s'", br.Type())
	}
	if br.IsReadOnly() {
		t.Error("Expected bedrock to be writable")
	}
	if !br.SupportsWatch() {
		t.Error("Expected bedrock to support watching")
	}

	// Start the bedrock
	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	// Test write
	testContent := []byte("Hello, Bedrock!")
	if err := br.Write("test.txt", testContent); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Test read
	content, err := br.Read("test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("Expected content '%s', got '%s'", testContent, content)
	}

	// Test stat
	info, err := br.Stat("test.txt")
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got '%s'", info.Name)
	}
	if info.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
	}

	// Test list
	files, err := br.List("")
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	// Test tree
	tree, err := br.Tree(5)
	if err != nil {
		t.Fatalf("Failed to get tree: %v", err)
	}
	if tree == "" {
		t.Error("Expected non-empty tree")
	}

	// Test delete
	if err := br.Delete("test.txt"); err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Verify deleted
	if _, err := br.Read("test.txt"); err != ErrNotFound {
		t.Error("Expected ErrNotFound after delete")
	}
}

func TestUnixBedrock_ReadOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bedrock-test-readonly-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name:     "test-readonly",
		Path:     tmpDir,
		ReadOnly: true,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	if !br.IsReadOnly() {
		t.Error("Expected bedrock to be read-only")
	}

	// Write should fail
	if err := br.Write("test.txt", []byte("test")); err != ErrReadOnly {
		t.Errorf("Expected ErrReadOnly, got %v", err)
	}
}

func TestUnixBedrock_PathValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bedrock-test-path-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name: "test",
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	// Path traversal should fail
	testCases := []string{
		"../escape",
		"foo/../../../etc/passwd",
		"/absolute/path",
	}

	for _, path := range testCases {
		if err := br.Write(path, []byte("test")); err != ErrInvalidPath {
			t.Errorf("Expected ErrInvalidPath for path '%s', got %v", path, err)
		}
	}
}

func TestUnixBedrock_NestedDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bedrock-test-nested-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name: "test",
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	// Write to nested path (should create directories)
	nestedPath := "a/b/c/test.txt"
	if err := br.Write(nestedPath, []byte("nested content")); err != nil {
		t.Fatalf("Failed to write nested file: %v", err)
	}

	// Verify we can read it
	content, err := br.Read(nestedPath)
	if err != nil {
		t.Fatalf("Failed to read nested file: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("Unexpected content: %s", content)
	}

	// Verify tree shows nested structure
	tree, err := br.Tree(10)
	if err != nil {
		t.Fatalf("Failed to get tree: %v", err)
	}
	t.Logf("Tree:\n%s", tree)
}

func TestUnixBedrock_Move(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bedrock-test-move-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name: "test",
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	// Create a file
	if err := br.Write("original.txt", []byte("test content")); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Move it
	if err := br.Move("original.txt", "moved.txt"); err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	// Verify original is gone
	if _, err := br.Read("original.txt"); err != ErrNotFound {
		t.Error("Expected original file to not exist")
	}

	// Verify new location has content
	content, err := br.Read("moved.txt")
	if err != nil {
		t.Fatalf("Failed to read moved file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Unexpected content: %s", content)
	}
}

func TestUnixBedrock_Watch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bedrock-test-watch-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name: "test",
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	// Set up event channel
	eventChan := make(chan BedrockEvent, 10)
	if err := br.Watch(context.Background(), func(e BedrockEvent) {
		eventChan <- e
	}); err != nil {
		t.Fatalf("Failed to set up watch: %v", err)
	}

	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	// Wait for mounted event
	select {
	case e := <-eventChan:
		if e.Type != BedrockEventMounted {
			t.Errorf("Expected mounted event, got %s", e.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for mounted event")
	}

	// Create a file directly (not through bedrock API to trigger fsnotify)
	testFile := filepath.Join(tmpDir, "watch-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Wait for create event
	select {
	case e := <-eventChan:
		if e.Type != BedrockEventFileCreated {
			t.Errorf("Expected file.created event, got %s", e.Type)
		}
		if e.Path != "watch-test.txt" {
			t.Errorf("Expected path 'watch-test.txt', got '%s'", e.Path)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for file created event")
	}
}

func TestUnixBedrock_Manifest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bedrock-test-manifest-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	br, err := NewUnixBedrock(UnixBedrockConfig{
		Name: "test-manifest",
		Path: tmpDir,
	})
	if err != nil {
		t.Fatalf("Failed to create bedrock: %v", err)
	}

	ctx := context.Background()
	if err := br.Start(ctx); err != nil {
		t.Fatalf("Failed to start bedrock: %v", err)
	}
	defer br.Stop()

	// Create some files
	for i := 0; i < 5; i++ {
		filename := filepath.Join("test", "file"+string(rune('a'+i))+".txt")
		if err := br.Write(filename, []byte("content")); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Get manifest
	manifest, err := br.Manifest()
	if err != nil {
		t.Fatalf("Failed to get manifest: %v", err)
	}

	if manifest.Name != "test-manifest" {
		t.Errorf("Expected name 'test-manifest', got '%s'", manifest.Name)
	}
	if manifest.Type != "unix" {
		t.Errorf("Expected type 'unix', got '%s'", manifest.Type)
	}
	if manifest.FileCount != 5 {
		t.Errorf("Expected 5 files, got %d", manifest.FileCount)
	}
	if manifest.TotalSize != 5*7 { // 5 files * len("content")
		t.Errorf("Expected total size %d, got %d", 5*7, manifest.TotalSize)
	}
}

func TestBedrockConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid unix bedrock",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"scratch": {Type: "unix", Path: "/tmp/scratch"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid git bedrock",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"docs": {Type: "git", Path: "/repos/docs", WriteMode: "commit"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid git bedrock with PR mode",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"docs": {
						Type:      "git",
						Path:      "/repos/docs",
						WriteMode: "pull_request",
						PRConfig: &BedrockPRConfig{
							BaseBranch:   "main",
							BranchPrefix: "nim/",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing type",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"invalid": {Path: "/tmp"},
				},
			},
			wantErr: true,
		},
		{
			name: "unix missing path",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"invalid": {Type: "unix"},
				},
			},
			wantErr: true,
		},
		{
			name: "git invalid write mode",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"invalid": {Type: "git", Path: "/tmp", WriteMode: "invalid"},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown type",
			config: Config{
				Bedrocks: map[string]BedrockConfig{
					"invalid": {Type: "unknown", Path: "/tmp"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBedrockLock_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		lock    BedrockLock
		expired bool
	}{
		{
			name: "pending PR never expires",
			lock: BedrockLock{
				Type:     LockTypePendingPR,
				Acquired: time.Now().Add(-24 * time.Hour),
				TTL:      30,
			},
			expired: false,
		},
		{
			name: "write lock not expired",
			lock: BedrockLock{
				Type:     LockTypeWrite,
				Acquired: time.Now(),
				TTL:      30,
			},
			expired: false,
		},
		{
			name: "write lock expired by TTL",
			lock: BedrockLock{
				Type:     LockTypeWrite,
				Acquired: time.Now().Add(-60 * time.Second),
				TTL:      30,
			},
			expired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.lock.IsExpired(); got != tt.expired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expired)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatSize(%d) = %s, want %s", tt.bytes, got, tt.expected)
			}
		})
	}
}
