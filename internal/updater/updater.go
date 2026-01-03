package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yourusername/nimsforest/internal/httputil"
	"github.com/yourusername/nimsforest/internal/updater/version"
)

const (
	// GitHubOwner is the GitHub organization/user that owns the repo
	GitHubOwner = "yourusername"
	// GitHubRepo is the repository name
	GitHubRepo = "nimsforest"
	// BinaryName is the name of the binary
	BinaryName = "forest"
)

var (
	githubAPIURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)
)

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// UpdateInfo contains information about an available update
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	UpdateURL      string
	ReleaseNotes   string
	Available      bool
}

// Updater handles version checking and updates
type Updater struct {
	currentVersion string
}

// NewUpdater creates a new Updater instance
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
	}
}

// CheckForUpdate checks if a new version is available using native HTTP client
func (u *Updater) CheckForUpdate() (*UpdateInfo, error) {
	// Create HTTP client with timeout and proper TLS configuration
	client := httputil.CreateHTTPClient(30 * time.Second)

	// Create request
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "nimsforest-updater")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	// Remove 'v' prefix if present
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(u.currentVersion, "v")

	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		UpdateURL:      release.HTMLURL,
		ReleaseNotes:   release.Body,
		Available:      version.Compare(latestVersion, currentVersion) > 0,
	}

	return info, nil
}

// PerformUpdate downloads and installs the latest version
func (u *Updater) PerformUpdate() error {
	// Get update info first to know which version to download
	updateInfo, err := u.CheckForUpdate()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !updateInfo.Available {
		fmt.Println("✅ Already on the latest version!")
		return nil
	}

	// Get the path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks if any
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	// Determine platform and architecture
	platform := httputil.GetPlatform()
	binaryName := fmt.Sprintf("%s-%s-%s", BinaryName, runtime.GOOS, runtime.GOARCH)

	// Construct download URL
	versionTag := "v" + updateInfo.LatestVersion
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		GitHubOwner, GitHubRepo, versionTag, binaryName)

	fmt.Printf("📦 Downloading %s %s for %s...\n", BinaryName, versionTag, platform)

	// Download binary to temporary file
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, BinaryName+"-update")

	if err := downloadFile(downloadURL, tmpFile); err != nil {
		return fmt.Errorf("failed to download binary: %w\n\nFallback: You can manually download from:\n%s", err, updateInfo.UpdateURL)
	}

	// Verify downloaded file is not empty
	fileInfo, err := os.Stat(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to check downloaded file: %w", err)
	}
	if fileInfo.Size() == 0 {
		os.Remove(tmpFile)
		return fmt.Errorf("downloaded file is empty")
	}

	// Make it executable
	if err := os.Chmod(tmpFile, 0755); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Verify the binary works (skip verification on platforms where exec may fail)
	fmt.Println("🔍 Verifying downloaded binary...")
	if !httputil.IsRestrictedEnvironment() {
		verifyCmd := exec.Command(tmpFile, "version")
		if output, err := verifyCmd.CombinedOutput(); err != nil {
			os.Remove(tmpFile)
			return fmt.Errorf("downloaded binary verification failed: %w\nOutput: %s", err, string(output))
		}
	} else {
		fmt.Println("⚠️  Skipping verification on restricted environment (Termux/Android)")
	}

	// Backup current version
	backupPath := execPath + ".backup"
	fmt.Printf("📋 Backing up current version to %s\n", backupPath)

	// Remove old backup if it exists
	os.Remove(backupPath)

	// Rename current binary to backup (this works even if the binary is running)
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to backup current version: %w", err)
	}

	// Replace current binary with new one using atomic rename
	fmt.Printf("✨ Installing update to %s\n", execPath)
	if err := os.Rename(tmpFile, execPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, execPath)
		os.Remove(tmpFile)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Clean up temporary file
	os.Remove(tmpFile)

	fmt.Println("\n✅ Update completed successfully!")
	fmt.Printf("\nRun '%s version' to verify the update.\n", BinaryName)
	fmt.Printf("Backup of previous version saved at: %s\n", backupPath)

	return nil
}

// downloadFile downloads a file from a URL to a local path using native HTTP client
func downloadFile(url, filepath string) error {
	// Create HTTP client with timeout and proper TLS configuration
	client := httputil.CreateHTTPClient(5 * time.Minute) // Longer timeout for binary downloads

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "nimsforest-updater")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// PrintUpdateInfo displays update information in a user-friendly format
func PrintUpdateInfo(info *UpdateInfo) {
	if info.Available {
		fmt.Printf("\n🆕 A new version is available!\n")
		fmt.Printf("   Current version: %s\n", info.CurrentVersion)
		fmt.Printf("   Latest version:  %s\n", info.LatestVersion)
		fmt.Printf("   Download URL:    %s\n", info.UpdateURL)
		if info.ReleaseNotes != "" {
			fmt.Printf("\n📝 Release Notes:\n%s\n", info.ReleaseNotes)
		}
		fmt.Printf("\n💡 To update, run: %s update\n", BinaryName)
	} else {
		fmt.Printf("✅ You're running the latest version (%s)\n", info.CurrentVersion)
	}
}
