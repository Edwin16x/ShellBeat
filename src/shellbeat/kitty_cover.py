"""
kitty_cover.py — protocolo gráfico de Kitty, escribe a /dev/tty
para funcionar dentro de Textual (que captura stdout).
"""
import base64
import os
from typing import Optional


def _encode_png(path: str) -> bytes:
    from PIL import Image
    import io
    with Image.open(path) as img:
        img = img.convert("RGBA")
        img = _autocrop_bars(img)
        buf = io.BytesIO()
        img.save(buf, format="PNG")
        return buf.getvalue()


def _autocrop_bars(img: "Image.Image") -> "Image.Image":
    """
    Recorta la imagen al cuadrado central.
    Las portadas de YouTube Music son 1280x720 (16:9) con barras laterales.
    Tomar el centro cuadrado (720x720) elimina esas barras.
    """
    w, h = img.size
    if w == h:
        return img  # ya es cuadrada, no hacer nada

    # Tomar el lado menor y recortar el centro
    side = min(w, h)
    left   = (w - side) // 2
    top    = (h - side) // 2
    right  = left + side
    bottom = top + side

    return img.crop((left, top, right, bottom))


def send_kitty_image(path: str, x: int, y: int, cols: int = 18, rows: int = 10) -> None:
    if not os.path.isfile(path):
        return
    try:
        img_bytes = _encode_png(path)
    except Exception:
        return

    payload = base64.standard_b64encode(img_bytes).decode("ascii")
    chunks = [payload[i:i + 4096] for i in range(0, len(payload), 4096)]

    # Escribimos directo a /dev/tty — bypasea el stdout de Textual
    with open("/dev/tty", "w") as tty:
        # Mover cursor a posición (1-indexed)
        tty.write(f"\033[{y + 1};{x + 1}H")

        for i, chunk in enumerate(chunks):
            m = 0 if i == len(chunks) - 1 else 1
            if i == 0:
                header = f"a=T,f=100,c={cols},r={rows},m={m}"
            else:
                header = f"m={m}"
            tty.write(f"\033_G{header};{chunk}\033\\")

        tty.flush()


def clear_kitty_images() -> None:
    try:
        with open("/dev/tty", "w") as tty:
            tty.write("\033_Ga=d,d=A\033\\")
            tty.flush()
    except Exception:
        pass
