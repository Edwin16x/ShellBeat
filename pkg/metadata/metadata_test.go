package metadata

import (
	"testing"
)

func TestFormatCleanTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Song Title [PL8UeZt7lYRzRfVTTYIlPfJEsr__m4fEmy].opus", "Song Title"},
		{"Normal Song.mp3", "Normal Song"},
		{"Artist - Track [12345].flac", "Artist - Track"},
	}

	for _, tt := range tests {
		result := FormatCleanTitle(tt.input)
		if result != tt.expected {
			t.Errorf("FormatCleanTitle(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestParseLRCString(t *testing.T) {
	rawLRC := `
[00:12.30]First line of lyrics
[00:15.500]Second line of lyrics
[01:02.10]Third line of lyrics
`

	lyrics, err := ParseLRCString(rawLRC)
	if err != nil {
		t.Fatalf("ParseLRCString returned error: %v", err)
	}

	if len(lyrics) != 3 {
		t.Fatalf("expected 3 lyric lines, got %d", len(lyrics))
	}

	if lyrics[0].Timestamp != 12.3 {
		t.Errorf("expected timestamp 12.3, got %f", lyrics[0].Timestamp)
	}
	if lyrics[0].Text != "First line of lyrics" {
		t.Errorf("expected text 'First line of lyrics', got %q", lyrics[0].Text)
	}

	if lyrics[1].Timestamp != 15.5 {
		t.Errorf("expected timestamp 15.5, got %f", lyrics[1].Timestamp)
	}

	if lyrics[2].Timestamp != 62.1 {
		t.Errorf("expected timestamp 62.1, got %f", lyrics[2].Timestamp)
	}
}
