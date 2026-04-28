import bisect
import os
import asyncio
from typing import Dict, Optional

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.screen import ModalScreen
from textual.widgets import (
    Footer, Input, Label, ListItem, ListView, ProgressBar, Static,
)
from textual.containers import Vertical, Horizontal, ScrollableContainer
from textual import on, work

from .engine import MusicPlayer, MetadataExtractor, LibraryScanner
from .engine.db import ShellBeatDB

_PKG = os.path.dirname(__file__)
MUSIC_DIR = os.path.normpath(os.path.join(_PKG, "..", "..", "musica"))

LYRIC_CLS = ["lyric-near-1", "lyric-near-2", "lyric-near-3", "lyric-dim"]

COLOR_PRESETS = {
    "Morado":     "#7F77DD",
    "Azul":       "#5B8DEF",
    "Verde":      "#4ADE80",
    "Rojo":       "#EF4444",
    "Naranja":    "#F59E0B",
    "Rosa":       "#EC4899",
    "Cyan":       "#22D3EE",
    "Dorado":     "#FBBF24",
    "Lavanda":    "#A78BFA",
    "Esmeralda":  "#34D399",
    "Coral":      "#FB7185",
    "Lima":       "#A3E635",
}

REPEAT_ICONS = {"off": "───", "all": "🔁 all", "one": "🔂 one"}


# ══════════════════════════════════════════════════════════════════════════════
# Modales
# ══════════════════════════════════════════════════════════════════════════════

class ThemeScreen(ModalScreen[str]):
    BINDINGS = [Binding("escape", "dismiss", "Cerrar")]

    def compose(self) -> ComposeResult:
        with Vertical(classes="modal-dialog"):
            yield Label("Seleccionar color de acento", classes="modal-title")
            for name, color in COLOR_PRESETS.items():
                yield Label(f"  [{color}]■■■[/]  {name}", classes="modal-item", id=f"t-{name}")

    def on_click(self, event) -> None:
        for name, color in COLOR_PRESETS.items():
            w = self.query_one(f"#t-{name}", Label)
            if w.region.contains_point(event.screen_offset):
                self.dismiss(color)
                return


class PlaylistCreateScreen(ModalScreen[str]):
    BINDINGS = [Binding("escape", "dismiss", "Cerrar")]

    def compose(self) -> ComposeResult:
        with Vertical(classes="modal-dialog"):
            yield Label("Crear nueva playlist", classes="modal-title")
            yield Input(placeholder="Nombre de la playlist…", id="pl-name")

    @on(Input.Submitted, "#pl-name")
    def _submit(self, event: Input.Submitted) -> None:
        if event.value.strip():
            self.dismiss(event.value.strip())


class HistoryScreen(ModalScreen):
    BINDINGS = [Binding("escape", "dismiss", "Cerrar")]

    def __init__(self, items: list, **kw) -> None:
        super().__init__(**kw)
        self._items = items

    def compose(self) -> ComposeResult:
        with Vertical(classes="modal-dialog"):
            yield Label("Historial de reproducción", classes="modal-title")
            if not self._items:
                yield Label("  (vacío)", classes="modal-item")
            for item in self._items[:20]:
                name = os.path.basename(item["track_path"])
                name = os.path.splitext(name)[0]
                if name.endswith("]") and "[" in name:
                    name = name[:name.rfind("[")].strip()
                ts = item.get("played_at", "")[:16]
                yield Label(f"  {name}  [#555]{ts}[/]", classes="modal-item")


class InfoScreen(ModalScreen):
    BINDINGS = [Binding("escape", "dismiss", "Cerrar")]

    def __init__(self, data: dict, **kw) -> None:
        super().__init__(**kw)
        self._data = data

    def compose(self) -> ComposeResult:
        d = self._data
        with Vertical(classes="modal-dialog"):
            yield Label("Información de la pista", classes="modal-title")
            for key in ["title", "artist", "album", "date", "bitrate", "path"]:
                val = d.get(key, "—")
                yield Label(f"  [#888]{key}:[/]  {val}", classes="modal-item")


class PlaylistSelectScreen(ModalScreen[int]):
    BINDINGS = [Binding("escape", "dismiss", "Cerrar")]

    def __init__(self, playlists: list, **kw) -> None:
        super().__init__(**kw)
        self._playlists = playlists

    def compose(self) -> ComposeResult:
        with Vertical(classes="modal-dialog"):
            yield Label("Seleccionar playlist", classes="modal-title")
            yield Label("  [#888]Todas las pistas[/]", id="pl-all", classes="modal-item")
            for pl in self._playlists:
                yield Label(
                    f"  {pl['name']}", id=f"pl-{pl['id']}", classes="modal-item"
                )

    def on_click(self, event) -> None:
        w = self.query_one("#pl-all", Label)
        if w.region.contains_point(event.screen_offset):
            self.dismiss(-1)
            return
        for pl in self._playlists:
            w = self.query_one(f"#pl-{pl['id']}", Label)
            if w.region.contains_point(event.screen_offset):
                self.dismiss(pl["id"])
                return


# ══════════════════════════════════════════════════════════════════════════════
# App principal
# ══════════════════════════════════════════════════════════════════════════════

class ShellBeat(App):

    TITLE = "ShellBeat"
    CSS_PATH = "style.css"

    BINDINGS = [
        Binding("q",     "quit",            "Salir"),
        Binding("space", "toggle_playback", "Play/Pausa"),
        Binding("n",     "next_track",      "Siguiente"),
        Binding("p",     "previous_track",  "Anterior"),
        Binding("right", "seek_forward",    "+10s"),
        Binding("left",  "seek_backward",   "-10s"),
        Binding("plus",  "volume_up",       "Vol+"),
        Binding("minus", "volume_down",     "Vol-"),
        Binding("s",     "toggle_shuffle",  "Shuffle"),
        Binding("r",     "toggle_repeat",   "Repeat"),
        Binding("f",     "toggle_favorite", "Favorito"),
        Binding("t",     "open_theme",      "Tema"),
        Binding("i",     "show_info",       "Info"),
        Binding("h",     "show_history",    "Historial"),
        Binding("c",     "create_playlist", "Crear PL"),
        Binding("l",     "select_playlist", "Playlists"),
        Binding("a",     "add_to_queue",    "A cola"),
        Binding("slash", "focus_search",    "Buscar"),
    ]

    def __init__(self) -> None:
        super().__init__()
        self.db = ShellBeatDB()
        self.player = MusicPlayer()
        self.meta = MetadataExtractor()
        self._all_tracks: list[str] = []
        self._tracks: list[str] = []
        self._names: list[str] = []
        self._cur_meta: dict = {}
        self._lyrics: Dict[float, str] = {}
        self._lyr_ts: list[float] = []
        self._lyr_w: list[Static] = []
        self._lyr_idx: int = -1
        self._play_idx: int = -1
        self._accent: str = self.db.get("accent_color", "#7F77DD")

    # ── Layout ────────────────────────────────────────────────────────────────

    def compose(self) -> ComposeResult:
        with Horizontal(id="main-row"):
            with Vertical(id="sidebar"):
                with Horizontal(classes="panel-header"):
                    yield Label("  biblioteca", classes="ph-title")
                    yield Label("", id="track-count", classes="ph-badge")
                yield Input(placeholder="🔍 buscar…", id="search-input")
                yield ListView(id="track-list")

            with Vertical(id="player-view"):
                with Horizontal(classes="panel-header"):
                    yield Label("  ahora suena", classes="ph-title")
                    yield Label("", id="format-badge", classes="ph-badge")
                yield Static("Sin canción", id="now-playing")
                yield Static("", id="artist-album")
                yield ProgressBar(total=100, show_eta=False, id="progress-bar")
                yield Static("", id="time-info")
                with Horizontal(id="volume-row"):
                    yield Label("vol", id="vol-label")
                    yield Static("", id="vol-bar")
                    yield Label("100%", id="vol-pct")
                yield Static("", id="mode-line")
                with Horizontal(classes="queue-header"):
                    yield Label("  siguiente", classes="ph-title")
                    yield Label("", id="queue-badge", classes="ph-badge")
                with ScrollableContainer(id="queue-scroll"):
                    pass

            with Vertical(id="lyrics-panel"):
                with Horizontal(classes="panel-header"):
                    yield Label("  letra", classes="ph-title")
                    yield Label("", id="lyrics-badge", classes="ph-badge")
                with ScrollableContainer(id="lyrics-scroll"):
                    yield Static("sin canción", classes="lyrics-status")

        yield Footer()

    # ── Montaje ───────────────────────────────────────────────────────────────

    async def on_mount(self) -> None:
        self._all_tracks = LibraryScanner.scan_folder(MUSIC_DIR)
        self._tracks = list(self._all_tracks)
        self._populate_list(self._tracks)

        vol = int(self.db.get("default_volume", "100"))
        self.player.volume = vol

        self.player.load_playlist(self._tracks)
        self.player.on_track_end = self._on_track_ended
        self._preload_artists()
        self.set_interval(1, self._tick)
        self._apply_accent()

    def _populate_list(self, tracks: list[str]) -> None:
        lv = self.query_one("#track-list", ListView)
        lv.clear()
        self._names = []
        for path in tracks:
            name = os.path.splitext(os.path.basename(path))[0]
            if name.endswith("]") and "[" in name:
                name = name[:name.rfind("[")].strip()
            label = name if len(name) <= 26 else name[:24] + "…"
            self._names.append(label)
            lv.append(ListItem(
                Label(f"  {label}", classes="track-title"),
                Label("  …", classes="track-artist"),
            ))
        self.query_one("#track-count", Label).update(
            f"[{self._accent}]{len(tracks)}[/]"
        )

    @work(thread=True)
    def _preload_artists(self) -> None:
        artists: dict[int, str] = {}
        for i, path in enumerate(self._tracks):
            try:
                from mutagen import File as MF
                a = MF(path)
                if a and a.tags:
                    art = MetadataExtractor._get_tag(a.tags, "artist", "")
                    if art and art != "Artista Desconocido":
                        artists[i] = art[:24]
            except Exception:
                pass
        self.call_from_thread(self._apply_artists, artists)

    def _apply_artists(self, artists: dict[int, str]) -> None:
        items = list(self.query_one("#track-list", ListView).query("ListItem"))
        for i, art in artists.items():
            if i < len(items):
                try:
                    fav = self.db.is_favorite(self._tracks[i])
                    pre = "♥ " if fav else "  "
                    items[i].query_one(".track-artist", Label).update(f"{pre}{art}")
                except Exception:
                    pass

    # ── Búsqueda ──────────────────────────────────────────────────────────────

    def action_focus_search(self) -> None:
        self.query_one("#search-input", Input).focus()

    @on(Input.Changed, "#search-input")
    def _on_search(self, event: Input.Changed) -> None:
        q = event.value.strip().lower()
        if not q:
            filtered = list(self._all_tracks)
        else:
            filtered = [p for p in self._all_tracks if q in os.path.basename(p).lower()]
        self._tracks = filtered
        self._populate_list(filtered)
        self.player.load_playlist(filtered)
        self._play_idx = -1

    # ── Eventos ───────────────────────────────────────────────────────────────

    @on(ListView.Selected, "#track-list")
    def _on_select(self, event: ListView.Selected) -> None:
        idx = self.query_one("#track-list", ListView).index
        if idx is not None:
            self.player.play(idx)
            self._mark(idx)
            self._update_info()

    # ── Acciones de reproducción ──────────────────────────────────────────────

    def action_toggle_playback(self) -> None:
        if self._tracks: self.player.toggle()

    def action_next_track(self) -> None:
        if not self._tracks: return
        self.player.next()
        self._mark(self.player.current_index)
        self._update_info()

    def action_previous_track(self) -> None:
        if not self._tracks: return
        self.player.previous()
        self._mark(self.player.current_index)
        self._update_info()

    def action_seek_forward(self) -> None:
        if self._tracks:
            self.player.seek(min(self.player.position + 10, self.player.duration))

    def action_seek_backward(self) -> None:
        if self._tracks:
            self.player.seek(max(self.player.position - 10, 0))

    def action_volume_up(self) -> None:
        self.player.volume = self.player.volume + 5

    def action_volume_down(self) -> None:
        self.player.volume = self.player.volume - 5

    def action_toggle_shuffle(self) -> None:
        self.player.shuffle = not self.player.shuffle
        self._update_mode_line()

    def action_toggle_repeat(self) -> None:
        self.player.cycle_repeat()
        self._update_mode_line()

    def action_add_to_queue(self) -> None:
        idx = self.query_one("#track-list", ListView).index
        if idx is not None:
            self.player.add_to_queue(idx)
            self._update_queue()

    # ── Favoritos ─────────────────────────────────────────────────────────────

    def action_toggle_favorite(self) -> None:
        path = self.player.current_track_path
        if not path: return
        is_fav = self.db.toggle_favorite(path)
        icon = "♥" if is_fav else "♡"
        self.notify(f"{icon} {'Favorito añadido' if is_fav else 'Favorito quitado'}")

    # ── Modales ───────────────────────────────────────────────────────────────

    def action_open_theme(self) -> None:
        def on_result(color: str) -> None:
            self._accent = color
            self.db.set("accent_color", color)
            self._apply_accent()
            if self._play_idx >= 0:
                self._mark(self._play_idx)
        self.push_screen(ThemeScreen(), on_result)

    def action_show_info(self) -> None:
        if self._cur_meta:
            self.push_screen(InfoScreen(self._cur_meta))

    def action_show_history(self) -> None:
        history = self.db.get_history(20)
        self.push_screen(HistoryScreen(history))

    def action_create_playlist(self) -> None:
        def on_name(name: str) -> None:
            pl_id = self.db.create_playlist(name)
            for path in self._tracks:
                self.db.add_to_playlist(pl_id, path)
            self.notify(f"Playlist '{name}' creada ({len(self._tracks)} pistas)")
        self.push_screen(PlaylistCreateScreen(), on_name)

    def action_select_playlist(self) -> None:
        pls = self.db.get_playlists()
        def on_pick(pl_id: int) -> None:
            if pl_id == -1:
                tracks = list(self._all_tracks)
            else:
                tracks = self.db.get_playlist_tracks(pl_id)
                tracks = [t for t in tracks if os.path.isfile(t)]
            self._tracks = tracks
            self._populate_list(tracks)
            self.player.load_playlist(tracks)
            self._play_idx = -1
        self.push_screen(PlaylistSelectScreen(pls), on_pick)

    # ── Auto-avance ───────────────────────────────────────────────────────────

    def _on_track_ended(self) -> None:
        self.call_from_thread(self._advance)

    def _advance(self) -> None:
        if not self._tracks: return
        self.player.next()
        self._mark(self.player.current_index)
        self._update_info()

    # ── Pista activa ──────────────────────────────────────────────────────────

    def _mark(self, idx: int) -> None:
        items = list(self.query_one("#track-list", ListView).query("ListItem"))
        if 0 <= self._play_idx < len(items):
            items[self._play_idx].remove_class("is-playing")
            items[self._play_idx].query_one(".track-title", Label).update(
                f"  {self._names[self._play_idx]}"
            )
        if 0 <= idx < len(items):
            items[idx].add_class("is-playing")
            items[idx].query_one(".track-title", Label).update(
                f"[{self._accent}]● {self._names[idx]}[/]"
            )
        self._play_idx = idx
        self.query_one("#track-list", ListView).index = idx

    # ── Info ──────────────────────────────────────────────────────────────────

    def _update_info(self) -> None:
        path = self.player.current_track_path
        if not path: return
        data = self.meta.get_metadata(path)
        data["path"] = path
        self._cur_meta = data

        self.query_one("#now-playing", Static).update(
            f"[bold {self._accent}]  {data['title']}[/]"
        )
        parts = [p for p in [
            data.get("artist", ""),
            data.get("album", "") if data.get("album") != data["title"] else "",
            f"({data['date']})" if data.get("date") else "",
        ] if p]
        self.query_one("#artist-album", Static).update("  ·  ".join(parts))

        br = data.get("bitrate", "")
        self.query_one("#format-badge", Label).update(
            f"[{self._accent}]opus · {br}[/]" if br and br != "N/A"
            else f"[{self._accent}]opus[/]"
        )

        self.db.add_history(path)
        self._update_mode_line()
        self._update_queue()
        self._load_lyrics(path, data)

    # ── Acento global ─────────────────────────────────────────────────────────

    def _apply_accent(self) -> None:
        c = self._accent
        self.query_one("#progress-bar", ProgressBar).styles.color = c
        self.query_one("#sidebar", Vertical).styles.border_right = ("solid", c)
        self.query_one("#player-view", Vertical).styles.border_right = ("solid", c)

    # ── Modos ─────────────────────────────────────────────────────────────────

    def _update_mode_line(self) -> None:
        parts = []
        if self.player.shuffle:
            parts.append(f"[{self._accent}]🔀 shuffle[/]")
        rep = self.player.repeat_mode
        if rep != "off":
            parts.append(f"[{self._accent}]{REPEAT_ICONS[rep]}[/]")
        self.query_one("#mode-line", Static).update("  ".join(parts) if parts else "")

    # ── Cola ──────────────────────────────────────────────────────────────────

    @work(exclusive=True, group="queue")
    async def _update_queue(self) -> None:
        qs = self.query_one("#queue-scroll", ScrollableContainer)
        await qs.remove_children()
        upcoming = self.player.get_upcoming(8)
        if not upcoming:
            await qs.mount(Static("  (vacía)", classes="queue-item"))
            self.query_one("#queue-badge", Label).update("")
            return
        self.query_one("#queue-badge", Label).update(
            f"[{self._accent}]{len(upcoming)}[/]"
        )
        for i, idx in enumerate(upcoming):
            if 0 <= idx < len(self._tracks):
                name = os.path.splitext(os.path.basename(self._tracks[idx]))[0]
                if name.endswith("]") and "[" in name:
                    name = name[:name.rfind("[")].strip()
                name = name if len(name) <= 30 else name[:28] + "…"
                await qs.mount(
                    Static(f"  [{self._accent}]{i+1}.[/] {name}", classes="queue-item")
                )

    # ── Letras ────────────────────────────────────────────────────────────────

    @work(exclusive=True)
    async def _load_lyrics(self, path: str, data: dict) -> None:
        self._lyrics = {}
        self._lyr_ts = []
        self._lyr_w = []
        self._lyr_idx = -1

        view = self.query_one("#lyrics-scroll", ScrollableContainer)
        await view.remove_children()
        await view.mount(Static("  buscando letra…", classes="lyrics-status"))
        self.query_one("#lyrics-badge", Label).update(f"[#555]buscando…[/]")

        lyrics = MetadataExtractor.load_lyrics(path)
        if not lyrics:
            loop = asyncio.get_running_loop()
            lyrics = await loop.run_in_executor(
                None,
                lambda: MetadataExtractor.download_lyrics_sync(
                    path, data.get("title", ""), data.get("artist", ""),
                ),
            )

        await view.remove_children()
        if not lyrics:
            await view.mount(Static("  letra no encontrada", classes="lyrics-status"))
            self.query_one("#lyrics-badge", Label).update("")
            return

        self.query_one("#lyrics-badge", Label).update(
            f"[{self._accent}]sincronizada[/]"
        )
        self._lyrics = lyrics
        self._lyr_ts = sorted(lyrics.keys())
        widgets = [Static("", classes="lyric-padding")]
        for ts in self._lyr_ts:
            w = Static(lyrics[ts], classes="lyric-line lyric-dim")
            self._lyr_w.append(w)
            widgets.append(w)
        widgets.append(Static("", classes="lyric-padding"))
        await view.mount(*widgets)

    def _sync_lyrics(self, pos: float) -> None:
        if not self._lyr_ts: return
        idx = bisect.bisect_right(self._lyr_ts, pos) - 1
        active = idx if idx >= 0 else -1
        if active == self._lyr_idx: return
        self._lyr_idx = active
        all_c = ["lyric-active"] + LYRIC_CLS
        for i, w in enumerate(self._lyr_w):
            for c in all_c: w.remove_class(c)
            if active < 0:
                w.add_class("lyric-dim"); continue
            d = abs(i - active)
            if d == 0:
                w.add_class("lyric-active")
                w.styles.color = self._accent
            else:
                w.add_class(LYRIC_CLS[min(d - 1, 3)])
        if active >= 0:
            try:
                self.query_one("#lyrics-scroll", ScrollableContainer).scroll_to_widget(
                    self._lyr_w[active], animate=True, center=True
                )
            except Exception: pass

    # ── Tick ──────────────────────────────────────────────────────────────────

    def _tick(self) -> None:
        pos, dur = self.player.position, self.player.duration
        if dur > 0:
            pct = pos / dur
            self.query_one("#progress-bar", ProgressBar).update(
                progress=int(pct * 100)
            )
            self.query_one("#time-info", Static).update(
                f"  {_fmt(pos)}  ─────  {_fmt(dur)}"
            )
            self._sync_lyrics(pos)

            # Fallback: detectar fin de pista por posición
            # (por si eof-reached de mpv no dispara el evento)
            if pos >= dur - 0.5 and dur > 5:
                self._advance()
                return

        vol = self.player.volume
        filled = round(vol / 100 * 16)
        self.query_one("#vol-bar", Static).update(
            f"[{self._accent}]" + "▰" * filled + f"[/][#2a2a2a]" + "▱" * (16 - filled) + "[/]"
        )
        self.query_one("#vol-pct", Label).update(f"{vol}%")

    def on_unmount(self) -> None:
        self.db.set("default_volume", str(self.player.volume))
        self.db.close()
        self.player.stop()


def _fmt(s: float) -> str:
    i = int(s)
    return f"{i // 60}:{i % 60:02d}"
