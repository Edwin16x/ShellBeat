import random
from typing import Any, Callable, List, Optional

import mpv  # type: ignore


class MusicPlayer:
    """Motor de reproducción basado en mpv."""

    def __init__(self) -> None:
        self.player: Any = mpv.MPV(video=False, ytdl=False)
        self._playlist: list[str] = []
        self._current_index: int = 0
        self._shuffle: bool = False
        self._shuffle_order: list[int] = []
        self._shuffle_pos: int = 0
        self._repeat_mode: str = "all"  # "off" | "one" | "all"
        self._manual_queue: list[int] = []
        self.on_track_end: Optional[Callable[[], None]] = None

        # Detección robusta de fin de pista via property observer
        @self.player.property_observer("eof-reached")
        def _on_eof(name: str, value: Any) -> None:
            if value and self.on_track_end:
                self.on_track_end()

    # ── Playlist ──────────────────────────────────────────────────────────────

    def load_playlist(self, tracks: list[str]) -> None:
        self._playlist = tracks
        self._current_index = 0
        self._rebuild_shuffle()

    def play(self, index: Optional[int] = None) -> None:
        if not self._playlist:
            return
        if index is not None:
            self._current_index = self._clamp(index)
        self.player.play(self._playlist[self._current_index])

    def toggle(self) -> None:
        self.player.pause = not self.player.pause

    def stop(self) -> None:
        try:
            self.player.stop()
        except Exception:
            pass

    # ── Navegación ────────────────────────────────────────────────────────────

    def next(self) -> None:
        if not self._playlist:
            return
        if self._repeat_mode == "one":
            self.play()
            return
        if self._manual_queue:
            self.play(self._manual_queue.pop(0))
            return
        if self._shuffle:
            self._shuffle_pos += 1
            if self._shuffle_pos >= len(self._shuffle_order):
                if self._repeat_mode == "all":
                    self._rebuild_shuffle()
                    self._shuffle_pos = 0
                else:
                    return
            self.play(self._shuffle_order[self._shuffle_pos])
        else:
            nxt = self._current_index + 1
            if nxt >= len(self._playlist):
                if self._repeat_mode == "all":
                    nxt = 0
                else:
                    return
            self.play(nxt)

    def previous(self) -> None:
        if not self._playlist:
            return
        if self._shuffle:
            self._shuffle_pos = max(0, self._shuffle_pos - 1)
            self.play(self._shuffle_order[self._shuffle_pos])
        else:
            self.play((self._current_index - 1) % len(self._playlist))

    def seek(self, seconds: float) -> None:
        try:
            self.player.seek(seconds, reference="absolute")
        except Exception:
            pass

    # ── Cola manual ───────────────────────────────────────────────────────────

    def add_to_queue(self, index: int) -> None:
        if 0 <= index < len(self._playlist):
            self._manual_queue.append(index)

    def get_upcoming(self, count: int = 8) -> List[int]:
        """Retorna los índices de las próximas N pistas."""
        result: list[int] = list(self._manual_queue[:count])
        remaining = count - len(result)
        if remaining <= 0:
            return result[:count]
        if self._shuffle:
            start = self._shuffle_pos + 1
            for i in range(start, min(start + remaining, len(self._shuffle_order))):
                result.append(self._shuffle_order[i])
        else:
            for i in range(remaining):
                idx = self._current_index + 1 + i
                if self._repeat_mode == "all":
                    idx = idx % len(self._playlist)
                if 0 <= idx < len(self._playlist):
                    result.append(idx)
        return result[:count]

    # ── Shuffle ───────────────────────────────────────────────────────────────

    def _rebuild_shuffle(self) -> None:
        indices = list(range(len(self._playlist)))
        random.shuffle(indices)
        if self._current_index in indices:
            indices.remove(self._current_index)
            indices.insert(0, self._current_index)
        self._shuffle_order = indices
        self._shuffle_pos = 0

    @property
    def shuffle(self) -> bool:
        return self._shuffle

    @shuffle.setter
    def shuffle(self, value: bool) -> None:
        self._shuffle = value
        if value:
            self._rebuild_shuffle()
            if self._current_index in self._shuffle_order:
                self._shuffle_pos = self._shuffle_order.index(self._current_index)

    # ── Repeat ────────────────────────────────────────────────────────────────

    @property
    def repeat_mode(self) -> str:
        return self._repeat_mode

    @repeat_mode.setter
    def repeat_mode(self, value: str) -> None:
        if value in ("off", "one", "all"):
            self._repeat_mode = value

    def cycle_repeat(self) -> str:
        modes = ["off", "all", "one"]
        idx = modes.index(self._repeat_mode) if self._repeat_mode in modes else 0
        self._repeat_mode = modes[(idx + 1) % 3]
        return self._repeat_mode

    # ── Propiedades ───────────────────────────────────────────────────────────

    @property
    def position(self) -> float:
        return float(self.player.time_pos or 0.0)

    @property
    def duration(self) -> float:
        return float(self.player.duration or 0.0)

    @property
    def volume(self) -> int:
        try:
            return int(self.player.volume or 100)
        except Exception:
            return 100

    @volume.setter
    def volume(self, value: int) -> None:
        self.player.volume = max(0, min(150, value))

    @property
    def current_track_path(self) -> Optional[str]:
        if 0 <= self._current_index < len(self._playlist):
            return self._playlist[self._current_index]
        return None

    @property
    def current_index(self) -> int:
        return self._current_index

    @current_index.setter
    def current_index(self, value: int) -> None:
        self._current_index = self._clamp(value)

    def _clamp(self, index: int) -> int:
        if not self._playlist:
            return 0
        return max(0, min(index, len(self._playlist) - 1))

    def __del__(self) -> None:
        try:
            self.player.terminate()
        except Exception:
            pass
