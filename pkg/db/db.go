package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Playlist struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type HistoryItem struct {
	ID        int64     `json:"id"`
	TrackPath string    `json:"track_path"`
	PlayedAt  time.Time `json:"played_at"`
}

type DB struct {
	conn *sql.DB
}

var defaultConfigs = map[string]string{
	"accent_color":   "#7F77DD",
	"bg_color":       "#111111",
	"sidebar_bg":     "#151515",
	"default_volume": "100",
}

// New initializes the database connection and creates required tables.
func New() (*DB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".shellbeat")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "shellbeat.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

func (d *DB) initSchema() error {
	schema := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS config (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS playlists (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS playlist_tracks (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		playlist_id INTEGER NOT NULL,
		track_path  TEXT NOT NULL,
		position    INTEGER NOT NULL,
		FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS favorites (
		track_path TEXT PRIMARY KEY,
		added_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS play_history (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		track_path TEXT NOT NULL,
		played_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to execute schema initialization: %w", err)
	}

	for k, v := range defaultConfigs {
		_, err := d.conn.Exec(`INSERT OR IGNORE INTO config (key, value) VALUES (?, ?)`, k, v)
		if err != nil {
			return fmt.Errorf("failed to set default config %s: %w", k, err)
		}
	}

	return nil
}

// Config operations
func (d *DB) GetConfig(key, defaultValue string) string {
	var val string
	err := d.conn.QueryRow(`SELECT value FROM config WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return defaultValue
	}
	return val
}

func (d *DB) SetConfig(key, value string) error {
	_, err := d.conn.Exec(`INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)`, key, value)
	return err
}

// Playlist operations
func (d *DB) CreatePlaylist(name string) (int64, error) {
	res, err := d.conn.Exec(`INSERT INTO playlists (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetPlaylists() ([]Playlist, error) {
	rows, err := d.conn.Query(`SELECT id, name, created_at FROM playlists ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, nil
}

func (d *DB) AddToPlaylist(playlistID int64, trackPath string) error {
	var maxPos sql.NullInt64
	err := d.conn.QueryRow(`SELECT MAX(position) FROM playlist_tracks WHERE playlist_id = ?`, playlistID).Scan(&maxPos)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	pos := int64(1)
	if maxPos.Valid {
		pos = maxPos.Int64 + 1
	}

	_, err = d.conn.Exec(`INSERT INTO playlist_tracks (playlist_id, track_path, position) VALUES (?, ?, ?)`, playlistID, trackPath, pos)
	return err
}

func (d *DB) GetPlaylistTracks(playlistID int64) ([]string, error) {
	rows, err := d.conn.Query(`SELECT track_path FROM playlist_tracks WHERE playlist_id = ? ORDER BY position`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		tracks = append(tracks, path)
	}
	return tracks, nil
}

func (d *DB) DeletePlaylist(playlistID int64) error {
	_, err := d.conn.Exec(`DELETE FROM playlist_tracks WHERE playlist_id = ?`, playlistID)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(`DELETE FROM playlists WHERE id = ?`, playlistID)
	return err
}

// Favorites operations
func (d *DB) ToggleFavorite(trackPath string) (bool, error) {
	if d.IsFavorite(trackPath) {
		_, err := d.conn.Exec(`DELETE FROM favorites WHERE track_path = ?`, trackPath)
		return false, err
	}
	_, err := d.conn.Exec(`INSERT INTO favorites (track_path) VALUES (?)`, trackPath)
	return true, err
}

func (d *DB) IsFavorite(trackPath string) bool {
	var dummy int
	err := d.conn.QueryRow(`SELECT 1 FROM favorites WHERE track_path = ?`, trackPath).Scan(&dummy)
	return err == nil
}

// Play history operations
func (d *DB) AddHistory(trackPath string) error {
	_, err := d.conn.Exec(`INSERT INTO play_history (track_path) VALUES (?)`, trackPath)
	return err
}

func (d *DB) GetHistory(limit int) ([]HistoryItem, error) {
	rows, err := d.conn.Query(`SELECT id, track_path, played_at FROM play_history ORDER BY played_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []HistoryItem
	for rows.Next() {
		var item HistoryItem
		if err := rows.Scan(&item.ID, &item.TrackPath, &item.PlayedAt); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}
