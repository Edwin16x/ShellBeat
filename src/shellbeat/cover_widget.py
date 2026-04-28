"""
cover_widget.py — Placeholder visual para la portada del álbum.
Muestra un fondo oscuro con el símbolo ◈ centrado.
"""
import os
from typing import Optional
from textual.widget import Widget
from rich.text import Text


class CoverWidget(Widget):

    DEFAULT_CSS = """
    CoverWidget {
        width: 1fr;
        height: 100%;
        background: #1a1a1a;
    }
    """

    def __init__(self, **kwargs) -> None:
        super().__init__(**kwargs)
        self._cover_path: Optional[str] = None
        self._accent: str = "#7F77DD"

    def set_cover(self, path: Optional[str], accent: str = "#7F77DD") -> None:
        self._cover_path = path
        self._accent = accent
        self.refresh()

    def render(self) -> Text:
        w = max(1, self.size.width)
        h = max(1, self.size.height)

        if not self._cover_path or not os.path.isfile(self._cover_path):
            return self._placeholder(w, h)

        try:
            from PIL import Image

            pixel_h = h * 2

            with Image.open(self._cover_path) as img:
                iw, ih = img.size
                side = min(iw, ih)
                left = (iw - side) // 2
                top  = (ih - side) // 2
                img  = img.crop((left, top, left + side, top + side))
                img  = img.convert("RGB").resize((w, pixel_h), Image.LANCZOS)
                px   = img.load()

            text = Text(no_wrap=True, overflow="crop")
            for row in range(0, pixel_h - 1, 2):
                for col in range(w):
                    tr, tg, tb = px[col, row]
                    br, bg, bb = px[col, row + 1]
                    text.append(
                        "▀",
                        style=f"#{tr:02x}{tg:02x}{tb:02x} on #{br:02x}{bg:02x}{bb:02x}",
                    )
                if row + 2 < pixel_h:
                    text.append("\n")

            return text

        except Exception:
            return self._placeholder(w, h)

    def _placeholder(self, w: int, h: int) -> Text:
        text = Text(no_wrap=True)
        pad_top = max(0, h // 2 - 1)
        for _ in range(pad_top):
            text.append(" " * w + "\n")
        pad_l = max(0, (w - 1) // 2)
        text.append(" " * pad_l + "◈", style=f"bold {self._accent}")
        return text
