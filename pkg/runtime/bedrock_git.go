package runtime

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// GitBedrock provides git repository access as a bedrock.
// It extends UnixBedrock with git operations and PR workflow support.
type GitBedrockImpl struct {
	*UnixBedrock
	remote    string
	branch    string
	writeMode string // "commit" or "pull_request"
	prConfig  *PRConfig

	gitMu sync.Mutex // Serialize git operations
}

// PRConfig configures pull request creation.
type PRConfig struct {
	BaseBranch   string   `yaml:"base_branch"`
	BranchPrefix string   `yaml:"branch_prefix"`
	Reviewers    []string `yaml:"reviewers,omitempty"`
	Labels       []string `yaml:"labels,omitempty"`
}

// GitBedrockConfig configures a Git bedrock.
type GitBedrockConfig struct {
	Name      string    `yaml:"name"`
	Path      string    `yaml:"path"`
	Remote    string    `yaml:"remote,omitempty"`
	Branch    string    `yaml:"branch,omitempty"`
	WriteMode string    `yaml:"write_mode,omitempty"` // "commit" or "pull_request"
	PRConfig  *PRConfig `yaml:"pr_config,omitempty"`
	ReadOnly  bool      `yaml:"readonly,omitempty"`
}

// NewGitBedrock creates a new Git bedrock.
func NewGitBedrock(cfg GitBedrockConfig) (*GitBedrockImpl, error) {
	// Validate write mode
	if cfg.WriteMode == "" {
		cfg.WriteMode = "commit"
	}
	if cfg.WriteMode != "commit" && cfg.WriteMode != "pull_request" {
		return nil, fmt.Errorf("invalid write_mode: %s (use commit or pull_request)", cfg.WriteMode)
	}

	// PR mode requires pr_config
	if cfg.WriteMode == "pull_request" && cfg.PRConfig == nil {
		cfg.PRConfig = &PRConfig{
			BaseBranch:   "main",
			BranchPrefix: "nim/",
		}
	}

	// Default branch
	if cfg.Branch == "" {
		cfg.Branch = "main"
	}

	// Resolve path
	absPath, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if it's a git repository
	if _, err := os.Stat(filepath.Join(absPath, ".git")); err != nil {
		if os.IsNotExist(err) {
			// Try to initialize or clone
			if cfg.Remote != "" {
				if err := cloneRepo(cfg.Remote, absPath, cfg.Branch); err != nil {
					return nil, fmt.Errorf("failed to clone repository: %w", err)
				}
			} else {
				return nil, fmt.Errorf("path is not a git repository: %s", absPath)
			}
		} else {
			return nil, fmt.Errorf("failed to check .git directory: %w", err)
		}
	}

	// Create underlying Unix bedrock
	unix, err := NewUnixBedrock(UnixBedrockConfig{
		Name:     cfg.Name,
		Path:     absPath,
		ReadOnly: cfg.ReadOnly,
	})
	if err != nil {
		return nil, err
	}

	return &GitBedrockImpl{
		UnixBedrock: unix,
		remote:      cfg.Remote,
		branch:      cfg.Branch,
		writeMode:   cfg.WriteMode,
		prConfig:    cfg.PRConfig,
	}, nil
}

// Type returns "git".
func (b *GitBedrockImpl) Type() string {
	return "git"
}

// Start starts the bedrock, syncing with remote first.
func (b *GitBedrockImpl) Start(ctx context.Context) error {
	// Sync with remote if configured
	if b.remote != "" {
		if err := b.Sync(ctx); err != nil {
			log.Printf("[Bedrock:%s] Warning: failed to sync with remote: %v", b.name, err)
		}
	}

	return b.UnixBedrock.Start(ctx)
}

// WriteMode returns the write mode ("commit" or "pull_request").
func (b *GitBedrockImpl) WriteMode() string {
	return b.writeMode
}

// CurrentBranch returns the current branch name.
func (b *GitBedrockImpl) CurrentBranch() string {
	return b.branch
}

// Remote returns the remote URL.
func (b *GitBedrockImpl) Remote() string {
	return b.remote
}

// Sync fetches and pulls from the remote repository.
func (b *GitBedrockImpl) Sync(ctx context.Context) error {
	b.gitMu.Lock()
	defer b.gitMu.Unlock()

	if b.remote == "" {
		return nil
	}

	// Fetch from origin
	if _, err := b.git(ctx, "fetch", "origin", b.branch); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// Reset to origin/branch (fast-forward)
	if _, err := b.git(ctx, "reset", "--hard", fmt.Sprintf("origin/%s", b.branch)); err != nil {
		return fmt.Errorf("failed to reset to origin/%s: %w", b.branch, err)
	}

	log.Printf("[Bedrock:%s] Synced with origin/%s", b.name, b.branch)
	return nil
}

// Write writes content and optionally commits or creates a PR.
func (b *GitBedrockImpl) Write(path string, content []byte) error {
	// First write the file using Unix bedrock
	if err := b.UnixBedrock.Write(path, content); err != nil {
		return err
	}

	// If not in commit or PR mode, just return
	if b.writeMode == "" || b.readonly {
		return nil
	}

	return nil // Commit/PR happens in WriteAndCommit
}

// WriteAndCommit writes content and commits with a message.
func (b *GitBedrockImpl) WriteAndCommit(ctx context.Context, path string, content []byte, message string) error {
	b.gitMu.Lock()
	defer b.gitMu.Unlock()

	// Write the file
	if err := b.UnixBedrock.Write(path, content); err != nil {
		return err
	}

	// Stage the file
	if _, err := b.git(ctx, "add", path); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	// Commit
	if message == "" {
		message = fmt.Sprintf("Update %s", path)
	}
	if _, err := b.git(ctx, "commit", "-m", message); err != nil {
		// Check if nothing to commit
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("[Bedrock:%s] Committed: %s", b.name, message)

	// Push if remote is configured
	if b.remote != "" {
		if _, err := b.git(ctx, "push", "origin", b.branch); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
		log.Printf("[Bedrock:%s] Pushed to origin/%s", b.name, b.branch)
	}

	return nil
}

// CreatePR creates a pull request for the given changes.
func (b *GitBedrockImpl) CreatePR(ctx context.Context, branchName, title, body string) (string, error) {
	b.gitMu.Lock()
	defer b.gitMu.Unlock()

	if b.prConfig == nil {
		return "", fmt.Errorf("PR config not set")
	}

	baseBranch := b.prConfig.BaseBranch
	if baseBranch == "" {
		baseBranch = b.branch
	}

	// Create a new branch
	prBranch := b.prConfig.BranchPrefix + branchName
	if _, err := b.git(ctx, "checkout", "-b", prBranch); err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	// Push the branch
	if _, err := b.git(ctx, "push", "-u", "origin", prBranch); err != nil {
		// Switch back to main branch on failure
		b.git(ctx, "checkout", baseBranch)
		b.git(ctx, "branch", "-D", prBranch)
		return "", fmt.Errorf("failed to push branch: %w", err)
	}

	// Create PR using gh CLI
	args := []string{"pr", "create",
		"--base", baseBranch,
		"--head", prBranch,
		"--title", title,
		"--body", body,
	}

	// Add reviewers
	for _, reviewer := range b.prConfig.Reviewers {
		args = append(args, "--reviewer", reviewer)
	}

	// Add labels
	for _, label := range b.prConfig.Labels {
		args = append(args, "--label", label)
	}

	output, err := b.gh(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w", err)
	}

	prURL := strings.TrimSpace(output)

	// Switch back to main branch
	if _, err := b.git(ctx, "checkout", baseBranch); err != nil {
		log.Printf("[Bedrock:%s] Warning: failed to switch back to %s: %v", b.name, baseBranch, err)
	}

	log.Printf("[Bedrock:%s] Created PR: %s", b.name, prURL)
	return prURL, nil
}

// WriteWithPR writes content and creates a PR for review.
func (b *GitBedrockImpl) WriteWithPR(ctx context.Context, path string, content []byte, title, body string) (string, error) {
	b.gitMu.Lock()
	defer b.gitMu.Unlock()

	if b.prConfig == nil {
		return "", fmt.Errorf("PR config not set")
	}

	baseBranch := b.prConfig.BaseBranch
	if baseBranch == "" {
		baseBranch = b.branch
	}

	// Generate branch name from path
	safePath := strings.ReplaceAll(path, "/", "-")
	safePath = strings.ReplaceAll(safePath, ".", "-")
	branchName := fmt.Sprintf("update-%s-%d", safePath, time.Now().Unix())
	prBranch := b.prConfig.BranchPrefix + branchName

	// Create a new branch
	if _, err := b.git(ctx, "checkout", "-b", prBranch); err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	// Write the file (without lock since we already hold it)
	fullPath := filepath.Join(b.root, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		b.git(ctx, "checkout", baseBranch)
		b.git(ctx, "branch", "-D", prBranch)
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		b.git(ctx, "checkout", baseBranch)
		b.git(ctx, "branch", "-D", prBranch)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Stage and commit
	if _, err := b.git(ctx, "add", path); err != nil {
		b.git(ctx, "checkout", baseBranch)
		b.git(ctx, "branch", "-D", prBranch)
		return "", fmt.Errorf("failed to stage file: %w", err)
	}

	commitMsg := title
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Update %s", path)
	}
	if _, err := b.git(ctx, "commit", "-m", commitMsg); err != nil {
		b.git(ctx, "checkout", baseBranch)
		b.git(ctx, "branch", "-D", prBranch)
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	// Push the branch
	if _, err := b.git(ctx, "push", "-u", "origin", prBranch); err != nil {
		b.git(ctx, "checkout", baseBranch)
		b.git(ctx, "branch", "-D", prBranch)
		return "", fmt.Errorf("failed to push branch: %w", err)
	}

	// Create PR using gh CLI
	args := []string{"pr", "create",
		"--base", baseBranch,
		"--head", prBranch,
		"--title", title,
		"--body", body,
	}

	for _, reviewer := range b.prConfig.Reviewers {
		args = append(args, "--reviewer", reviewer)
	}
	for _, label := range b.prConfig.Labels {
		args = append(args, "--label", label)
	}

	output, err := b.gh(ctx, args...)
	if err != nil {
		// PR creation failed, but changes are pushed
		b.git(ctx, "checkout", baseBranch)
		return "", fmt.Errorf("failed to create PR (branch %s pushed): %w", prBranch, err)
	}

	prURL := strings.TrimSpace(output)

	// Switch back to main branch
	if _, err := b.git(ctx, "checkout", baseBranch); err != nil {
		log.Printf("[Bedrock:%s] Warning: failed to switch back to %s: %v", b.name, baseBranch, err)
	}

	log.Printf("[Bedrock:%s] Created PR: %s", b.name, prURL)
	return prURL, nil
}

// CheckPRStatus checks the status of a pull request.
func (b *GitBedrockImpl) CheckPRStatus(ctx context.Context, prURL string) (string, error) {
	// Extract PR number from URL
	parts := strings.Split(prURL, "/")
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid PR URL")
	}
	prNumber := parts[len(parts)-1]

	output, err := b.gh(ctx, "pr", "view", prNumber, "--json", "state", "--jq", ".state")
	if err != nil {
		return "", fmt.Errorf("failed to check PR status: %w", err)
	}

	return strings.TrimSpace(strings.ToLower(output)), nil
}

// Manifest returns the bedrock manifest with git info.
func (b *GitBedrockImpl) Manifest() (*BedrockManifest, error) {
	manifest, err := b.UnixBedrock.Manifest()
	if err != nil {
		return nil, err
	}

	manifest.Type = "git"
	manifest.Remote = b.remote
	manifest.Branch = b.branch

	return manifest, nil
}

// git executes a git command in the repository.
func (b *GitBedrockImpl) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = b.root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// gh executes a gh CLI command.
func (b *GitBedrockImpl) gh(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = b.root

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// cloneRepo clones a repository.
func cloneRepo(remote, path, branch string) error {
	args := []string{"clone", "--branch", branch, "--single-branch", remote, path}
	cmd := exec.Command("git", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", err, stderr.String())
	}

	return nil
}

// GetCommitHash returns the current commit hash.
func (b *GitBedrockImpl) GetCommitHash(ctx context.Context) (string, error) {
	output, err := b.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// GetFileHistory returns the git log for a file.
func (b *GitBedrockImpl) GetFileHistory(ctx context.Context, path string, limit int) (string, error) {
	if limit <= 0 {
		limit = 10
	}
	output, err := b.git(ctx, "log", fmt.Sprintf("-n%d", limit), "--oneline", "--", path)
	if err != nil {
		return "", err
	}
	return output, nil
}

// GetDiff returns the diff of uncommitted changes.
func (b *GitBedrockImpl) GetDiff(ctx context.Context) (string, error) {
	output, err := b.git(ctx, "diff", "HEAD")
	if err != nil {
		return "", err
	}
	return output, nil
}

// HasUncommittedChanges returns true if there are uncommitted changes.
func (b *GitBedrockImpl) HasUncommittedChanges(ctx context.Context) (bool, error) {
	output, err := b.git(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}
