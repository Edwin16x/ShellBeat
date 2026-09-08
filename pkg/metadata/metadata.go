package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type TrackInfo struct {
	FilePath   string  `json:"file_path"`
	FileName   string  `json:"file_name"`
	CleanTitle string  `json:"clean_title"`
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Album      string  `json:"album"`
	Year       string  `json:"year"`
	Bitrate    string  `json:"bitrate"`
	SampleRate string  `json:"sample_rate"`
	Duration   float64 `json:"duration"`
	CoverPath  string  `json:"cover_path"`
}

type LyricLine struct {
	Timestamp float64 `json:"timestamp"`
	Text      string  `json:"text"`
}

type MetadataExtractor struct{}

func New() *MetadataExtractor {
	return &MetadataExtractor{}
}

// FormatCleanTitle strips youtube-dl IDs like "Title [xyz123].ext" -> "Title"
func FormatCleanTitle(filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	if idx := strings.LastIndex(base, "["); idx != -1 && strings.HasSuffix(base, "]") {
		base = strings.TrimSpace(base[:idx])
	}
	return base
}

// GetMetadata extracts audio tag metadata from the given audio path.
func (m *MetadataExtractor) GetMetadata(path string) TrackInfo {
	baseName := filepath.Base(path)
	cleanTitle := FormatCleanTitle(baseName)

	info := TrackInfo{
		FilePath:   path,
		FileName:   baseName,
		CleanTitle: cleanTitle,
		Title:      cleanTitle,
		Artist:     "Artista Desconocido",
		Album:      "Álbum Desconocido",
		Year:       "—",
		Bitrate:    "—",
		SampleRate: "—",
	}

	file, err := os.Open(path)
	if err != nil {
		return info
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err == nil {
		if t := metadata.Title(); t != "" {
			info.Title = t
		}
		if a := metadata.Artist(); a != "" {
			info.Artist = a
		}
		if al := metadata.Album(); al != "" {
			info.Album = al
		}
		if y := metadata.Year(); y != 0 {
			info.Year = fmt.Sprintf("%d", y)
		}

		info.CoverPath = m.FindCover(path)
	}

	return info
}

// FindCover looks for associated cover art image file.
func (m *MetadataExtractor) FindCover(audioPath string) string {
	exts := []string{".webp", ".jpg", ".jpeg", ".png"}
	base := strings.TrimSuffix(audioPath, filepath.Ext(audioPath))
	for _, ext := range exts {
		candidate := base + ext
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// GetLRCPath returns expected path for .lrc lyrics file.
func (m *MetadataExtractor) GetLRCPath(audioPath string) string {
	return strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".lrc"
}

// LoadLyrics loads .lrc lyrics from disk if available.
func (m *MetadataExtractor) LoadLyrics(audioPath string) ([]LyricLine, bool) {
	lrcPath := m.GetLRCPath(audioPath)
	if _, err := os.Stat(lrcPath); err != nil {
		return nil, false
	}
	lyrics, err := ParseLRC(lrcPath)
	if err != nil || len(lyrics) == 0 {
		return nil, false
	}
	return lyrics, true
}

// ParseLRC parses a .lrc file into sorted LyricLines.
func ParseLRC(lrcPath string) ([]LyricLine, error) {
	content, err := os.ReadFile(lrcPath)
	if err != nil {
		return nil, err
	}

	return ParseLRCString(string(content))
}

var lrcRegex = regexp.MustCompile(`\[(\d{1,2}):(\d{2})\.(\d{2,3})\](.*)`)

// ParseLRCString parses raw LRC content string into sorted LyricLines.
func ParseLRCString(content string) ([]LyricLine, error) {
	lines := strings.Split(content, "\n")
	var lyrics []LyricLine

	for _, line := range lines {
		line = strings.TrimSpace(line)
		match := lrcRegex.FindStringSubmatch(line)
		if len(match) == 5 {
			min, _ := strconv.Atoi(match[1])
			sec, _ := strconv.Atoi(match[2])
			msRaw := match[3]
			text := strings.TrimSpace(match[4])

			msVal, _ := strconv.Atoi(msRaw)
			var ms float64
			if len(msRaw) == 3 {
				ms = float64(msVal) / 1000.0
			} else {
				ms = float64(msVal) / 100.0
			}

			ts := float64(min*60+sec) + ms
			if text != "" {
				lyrics = append(lyrics, LyricLine{
					Timestamp: ts,
					Text:      text,
				})
			}
		}
	}

	sort.Slice(lyrics, func(i, j int) bool {
		return lyrics[i].Timestamp < lyrics[j].Timestamp
	})

	return lyrics, nil
}

type LRCLibResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
}

// DownloadLyricsLRCLIB fetches synced lyrics from LRCLIB API and saves to disk.
func (m *MetadataExtractor) DownloadLyricsLRCLIB(audioPath, title, artist string) ([]LyricLine, error) {
	lrcPath := m.GetLRCPath(audioPath)

	apiURL := fmt.Sprintf("https://lrclib.net/api/get?track_name=%s&artist_name=%s",
		url.QueryEscape(title), url.QueryEscape(artist))

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res LRCLibResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	if res.SyncedLyrics == "" {
		return nil, fmt.Errorf("no synced lyrics found")
	}

	_ = os.WriteFile(lrcPath, []byte(res.SyncedLyrics), 0644)
	return ParseLRCString(res.SyncedLyrics)
}
