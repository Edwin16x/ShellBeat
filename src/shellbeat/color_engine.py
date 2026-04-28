"""
color_engine.py — extrae colores dominantes de una imagen
y genera paletas suavizadas para el tema de la app.
"""
import colorsys
from PIL import Image


def extract_colors(image_path: str, n: int = 2) -> list[tuple[int, int, int]]:
    """
    Extrae los N colores más dominantes y vivos de una imagen.
    Boost de saturación para imágenes oscuras como portadas de música.
    """
    PALETTE_SIZE = 32
    try:
        with Image.open(image_path) as img:
            img = img.convert("RGB").resize((200, 200))
            paletted = img.quantize(colors=PALETTE_SIZE, method=Image.Quantize.FASTOCTREE)
            palette  = paletted.getpalette()
            hist     = paletted.histogram()

            scored = []
            for i in range(PALETTE_SIZE):
                r, g, b = palette[i*3], palette[i*3+1], palette[i*3+2]
                count   = hist[i]
                h, s, v = colorsys.rgb_to_hsv(r/255, g/255, b/255)
                # Priorizamos saturación y que no sea ni negro ni blanco puro
                score = count * (s ** 0.4) * max(0, v - 0.05) * (1 - max(0, v - 0.92))
                scored.append((score, (r, g, b)))

            scored.sort(reverse=True)
            candidates = [c for _, c in scored if sum(c) > 30]  # descarta negros puros

            if len(candidates) < n:
                candidates += [(80, 60, 120)] * n

            return candidates[:n]
    except Exception:
        return [(80, 60, 120), (40, 30, 80)]


def ensure_visible(
    color: tuple[int, int, int],
    min_brightness: int = 60,
) -> tuple[int, int, int]:
    """
    Garantiza que un color tenga suficiente brillo para verse como fondo.
    Si es muy oscuro, lo lleva a HSV y sube el value mínimo.
    """
    r, g, b = color
    h, s, v = colorsys.rgb_to_hsv(r/255, g/255, b/255)
    # Forzamos un brillo mínimo y saturación mínima para que se note el color
    v = max(v, min_brightness / 255)
    s = max(s, 0.35)
    nr, ng, nb = colorsys.hsv_to_rgb(h, s, v)
    return (int(nr * 255), int(ng * 255), int(nb * 255))


def as_bg(color: tuple[int, int, int]) -> tuple[int, int, int]:
    """Versión oscura del color apta para fondo (no demasiado oscura)."""
    r, g, b = ensure_visible(color, min_brightness=55)
    h, s, v = colorsys.rgb_to_hsv(r/255, g/255, b/255)
    v = min(v, 0.30)   # cap de brillo para que sea un fondo oscuro pero visible
    s = max(s, 0.40)
    nr, ng, nb = colorsys.hsv_to_rgb(h, s, v)
    return (int(nr * 255), int(ng * 255), int(nb * 255))


def as_accent(color: tuple[int, int, int]) -> tuple[int, int, int]:
    """Versión brillante del color para acentos y texto secundario."""
    r, g, b = color
    h, s, v = colorsys.rgb_to_hsv(r/255, g/255, b/255)
    v = max(v, 0.70)
    s = max(s, 0.50)
    nr, ng, nb = colorsys.hsv_to_rgb(h, s, v)
    return (int(nr * 255), int(ng * 255), int(nb * 255))


def to_hex(color: tuple[int, int, int]) -> str:
    """Convierte una tupla RGB a string hexadecimal CSS."""
    return "#{:02x}{:02x}{:02x}".format(*color)
