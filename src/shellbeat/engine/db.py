"""
db.py — Base de datos SQLite para ShellBeat.
Almacena configuración, playlists, favoritos e historial.
Ubicación: ~/.shellbeat/shellbeat.db
"""
import sqlite3
import os
from typing import List, Dict


DB_DIR = os.path.expanduser("~/.shellbeat")
DB_PATH = os.path.join(DB_DIR, "shellbeat.db")

DEFAULTS = {
    "accent_color": "#7F77DD",
    "bg_color": "#111111",
    "sidebar_bg": "#151515",
    "default_volume": "100",
}


class ShellBeatDB:

    def __init__(self) -> None:
        os.makedirs(DB_DIR, exist_ok=True)
        self.conn = sqlite3.connect(DB_PATH)
        self.conn.row_factory = sqlite3.Row
        self.conn.execute("PRAGMA foreign_keys = ON")
        self._create_tables()

    def _create_tables(self) -> None:
        self.conn.executescript("""
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
                id        INTEGER PRIMARY KEY AUTOINCREMENT,
                track_path TEXT NOT NULL,
                played_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        """)
        for key, value in DEFAULTS.items():
            self.conn.execute(
                "INSERT OR IGNORE INTO config (key, value) VALUES (?, ?)",
                (key, value),
            )
        self.conn.commit()

    # ── Config ────────────────────────────────────────────────────────────────

    def get(self, key: str, default: str = "") -> str:
        row = self.conn.execute(
            "SELECT value FROM config WHERE key=?", (key,)
        ).fetchone()
        return row["value"] if row else default

    def set(self, key: str, value: str) -> None:
        self.conn.execute(
            "INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)",
            (key, value),
        )
        self.conn.commit()

    # ── Playlists ─────────────────────────────────────────────────────────────

    def create_playlist(self, name: str) -> int:
        cur = self.conn.execute(
            "INSERT INTO playlists (name) VALUES (?)", (name,)
        )
        self.conn.commit()
        return cur.lastrowid  # type: ignore

    def get_playlists(self) -> List[Dict]:
        rows = self.conn.execute(
            "SELECT id, name, created_at FROM playlists ORDER BY name"
        ).fetchall()
        return [dict(r) for r in rows]

    def add_to_playlist(self, playlist_id: int, track_path: str) -> None:
        pos = self.conn.execute(
            "SELECT COALESCE(MAX(position),0)+1 FROM playlist_tracks WHERE playlist_id=?",
            (playlist_id,),
        ).fetchone()[0]
        self.conn.execute(
            "INSERT INTO playlist_tracks (playlist_id, track_path, position) VALUES (?,?,?)",
            (playlist_id, track_path, pos),
        )
        self.conn.commit()

    def get_playlist_tracks(self, playlist_id: int) -> List[str]:
        rows = self.conn.execute(
            "SELECT track_path FROM playlist_tracks WHERE playlist_id=? ORDER BY position",
            (playlist_id,),
        ).fetchall()
        return [r["track_path"] for r in rows]

    def delete_playlist(self, playlist_id: int) -> None:
        self.conn.execute("DELETE FROM playlist_tracks WHERE playlist_id=?", (playlist_id,))
        self.conn.execute("DELETE FROM playlists WHERE id=?", (playlist_id,))
        self.conn.commit()

    # ── Favoritos ─────────────────────────────────────────────────────────────

    def toggle_favorite(self, track_path: str) -> bool:
        """Retorna True si quedó como favorito, False si se quitó."""
        exists = self.conn.execute(
            "SELECT 1 FROM favorites WHERE track_path=?", (track_path,)
        ).fetchone()
        if exists:
            self.conn.execute("DELETE FROM favorites WHERE track_path=?", (track_path,))
            self.conn.commit()
            return False
        self.conn.execute("INSERT INTO favorites (track_path) VALUES (?)", (track_path,))
        self.conn.commit()
        return True

    def is_favorite(self, track_path: str) -> bool:
        return self.conn.execute(
            "SELECT 1 FROM favorites WHERE track_path=?", (track_path,)
        ).fetchone() is not None

    # ── Historial ─────────────────────────────────────────────────────────────

    def add_history(self, track_path: str) -> None:
        self.conn.execute(
            "INSERT INTO play_history (track_path) VALUES (?)", (track_path,)
        )
        self.conn.commit()

    def get_history(self, limit: int = 50) -> List[Dict]:
        rows = self.conn.execute(
            "SELECT track_path, played_at FROM play_history ORDER BY played_at DESC LIMIT ?",
            (limit,),
        ).fetchall()
        return [dict(r) for r in rows]

    def close(self) -> None:
        self.conn.close()
