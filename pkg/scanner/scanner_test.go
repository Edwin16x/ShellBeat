package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFolder(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "shellbeat_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create dummy files
	f1 := filepath.Join(tempDir, "song1.opus")
	f2 := filepath.Join(tempDir, "song2.flac")
	f3 := filepath.Join(tempDir, "readme.txt")

	_ = os.WriteFile(f1, []byte("dummy audio"), 0644)
	_ = os.WriteFile(f2, []byte("dummy audio"), 0644)
	_ = os.WriteFile(f3, []byte("not audio"), 0644)

	s := New()
	tracks, err := s.ScanFolder(tempDir)
	if err != nil {
		t.Fatalf("ScanFolder error: %v", err)
	}

	if len(tracks) != 2 {
		t.Fatalf("expected 2 audio tracks, got %d", len(tracks))
	}

	if tracks[0] != f1 || tracks[1] != f2 {
		t.Errorf("unexpected tracks returned: %v", tracks)
	}
}
