package updater

import (
	"testing"
)

func TestNewUpdater(t *testing.T) {
	u := NewUpdater("1.0.0")

	if u == nil {
		t.Fatal("NewUpdater returned nil")
	}

	if u.currentVersion != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", u.currentVersion)
	}
}

func TestNewUpdater_DevVersion(t *testing.T) {
	u := NewUpdater("dev")

	if u == nil {
		t.Fatal("NewUpdater returned nil")
	}

	if u.currentVersion != "dev" {
		t.Errorf("Expected version dev, got %s", u.currentVersion)
	}
}

func TestNewUpdater_WithVPrefix(t *testing.T) {
	u := NewUpdater("v1.2.3")

	if u == nil {
		t.Fatal("NewUpdater returned nil")
	}

	if u.currentVersion != "v1.2.3" {
		t.Errorf("Expected version v1.2.3, got %s", u.currentVersion)
	}
}

func TestUpdateInfoStruct(t *testing.T) {
	info := &UpdateInfo{
		CurrentVersion: "1.0.0",
		LatestVersion:  "1.1.0",
		UpdateURL:      "https://example.com/release",
		ReleaseNotes:   "Bug fixes",
		Available:      true,
	}

	if info.CurrentVersion != "1.0.0" {
		t.Errorf("Expected CurrentVersion 1.0.0, got %s", info.CurrentVersion)
	}

	if info.LatestVersion != "1.1.0" {
		t.Errorf("Expected LatestVersion 1.1.0, got %s", info.LatestVersion)
	}

	if !info.Available {
		t.Error("Expected Available to be true")
	}
}

func TestGitHubReleaseStruct(t *testing.T) {
	release := GitHubRelease{
		TagName: "v1.0.0",
		Name:    "Release v1.0.0",
		Body:    "Release notes",
		HTMLURL: "https://github.com/repo/releases/v1.0.0",
	}

	if release.TagName != "v1.0.0" {
		t.Errorf("Expected TagName v1.0.0, got %s", release.TagName)
	}

	if release.Name != "Release v1.0.0" {
		t.Errorf("Expected Name 'Release v1.0.0', got %s", release.Name)
	}
}

// Note: CheckForUpdate and PerformUpdate are integration tests
// that would require network access to GitHub API.
// They should be tested in a CI environment with proper mocking
// or in integration test files.
