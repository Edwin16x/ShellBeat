package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
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

	sort.Slice(tracks, func(i, j int) bool {
		return naturalLess(tracks[i], tracks[j])
	})
	return tracks, nil
}

func naturalLess(s1, s2 string) bool {
	i, j := 0, 0
	len1, len2 := len(s1), len(s2)

	for i < len1 && j < len2 {
		r1, size1 := utf8.DecodeRuneInString(s1[i:])
		r2, size2 := utf8.DecodeRuneInString(s2[j:])

		isDigit1 := unicode.IsDigit(r1)
		isDigit2 := unicode.IsDigit(r2)

		if isDigit1 && isDigit2 {
			start1 := i
			for i < len1 {
				r, sz := utf8.DecodeRuneInString(s1[i:])
				if !unicode.IsDigit(r) {
					break
				}
				i += sz
			}
			start2 := j
			for j < len2 {
				r, sz := utf8.DecodeRuneInString(s2[j:])
				if !unicode.IsDigit(r) {
					break
				}
				j += sz
			}

			d1 := strings.TrimLeft(s1[start1:i], "0")
			d2 := strings.TrimLeft(s2[start2:j], "0")

			if len(d1) != len(d2) {
				return len(d1) < len(d2)
			}
			if d1 != d2 {
				return d1 < d2
			}
			if (i - start1) != (j - start2) {
				return (i - start1) < (j - start2)
			}
			continue
		}

		c1 := unicode.ToLower(r1)
		c2 := unicode.ToLower(r2)
		if c1 != c2 {
			return c1 < c2
		}

		i += size1
		j += size2
	}

	return len1 < len2
}
