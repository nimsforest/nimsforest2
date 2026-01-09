package songbirds

import (
	"testing"
)

func TestBaseSongbird(t *testing.T) {
	t.Run("NewBaseSongbird sets fields correctly", func(t *testing.T) {
		sb := NewBaseSongbird("test-songbird", "song.test.>", nil)

		if sb.Name() != "test-songbird" {
			t.Errorf("Name() = %s, want test-songbird", sb.Name())
		}

		if sb.Pattern() != "song.test.>" {
			t.Errorf("Pattern() = %s, want song.test.>", sb.Pattern())
		}

		if sb.IsRunning() {
			t.Error("IsRunning() should be false initially")
		}
	})

	t.Run("SetRunning updates state", func(t *testing.T) {
		sb := NewBaseSongbird("test", "song.>", nil)

		sb.SetRunning(true)
		if !sb.IsRunning() {
			t.Error("IsRunning() should be true after SetRunning(true)")
		}

		sb.SetRunning(false)
		if sb.IsRunning() {
			t.Error("IsRunning() should be false after SetRunning(false)")
		}
	})

	t.Run("GetWind returns wind reference", func(t *testing.T) {
		sb := NewBaseSongbird("test", "song.>", nil)

		if sb.GetWind() != nil {
			t.Error("GetWind() should return nil when no wind set")
		}
	})
}
