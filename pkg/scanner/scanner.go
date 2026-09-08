package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var AudioExtensions = map[string]bool{
	".mp3":  true,
	".flac": true,
	".ogg":  true,
	".opus": true,
	".wav":  true,
	".m4a":  true,
	".aac":  true,
	".wv":   true,
}

type Scanner struct{}

func New() *Scanner {
	return &Scanner{}
}

// ScanFolder returns a sorted list of audio file paths within folder.
func (s *Scanner) ScanFolder(folder string) ([]string, error) {
	var tracks []string

	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if AudioExtensions[ext] {
			tracks = append(tracks, path)
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Strings(tracks)
	return tracks, nil
}
