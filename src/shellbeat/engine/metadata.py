import io
import os
import re
from typing import Any, Dict, Optional
from mutagen import File as MutagenFile  # type: ignore

AUDIO_EXTENSIONS = {".mp3", ".flac", ".ogg", ".opus", ".wav", ".m4a", ".aac", ".wv"}
COVER_EXTENSIONS = [".webp", ".jpg", ".jpeg", ".png"]


class MetadataExtractor:
    """
    Se encarga de leer tags de archivos de audio,
    localizar portadas y gestionar letras sincronizadas (.lrc).
    """

    @staticmethod
    def get_metadata(path: str) -> Dict[str, Any]:
        base_name = os.path.basename(path)

        try:
            audio = MutagenFile(path)
        except Exception as e:
            return {"file": base_name, "error": str(e)}

        if audio is None:
            return {"file": base_name, "error": "Formato no soportado"}

        metadata: Dict[str, Any] = {
            "file":        base_name,
            "title":       os.path.splitext(base_name)[0],
            "artist":      "Artista Desconocido",
            "album":       "Álbum Desconocido",
            "date":        "",
            "bitrate":     "N/A",
            "sample_rate": "N/A",
            "cover_path":  MetadataExtractor.find_cover(path),
        }

        info = getattr(audio, "info", None)
        if info:
            if hasattr(info, "bitrate") and info.bitrate:
                metadata["bitrate"] = f"{int(info.bitrate / 1000)} kbps"
            if hasattr(info, "sample_rate") and info.sample_rate:
                metadata["sample_rate"] = f"{info.sample_rate / 1000:.1f} kHz"

        tags = getattr(audio, "tags", None) or {}
        metadata["title"]  = MetadataExtractor._get_tag(tags, "title",  metadata["title"])
        metadata["artist"] = MetadataExtractor._get_tag(tags, "artist", metadata["artist"])
        metadata["album"]  = MetadataExtractor._get_tag(tags, "album",  metadata["album"])

        raw_date = MetadataExtractor._get_tag(tags, "date", "")
        metadata["date"] = raw_date[:4] if raw_date else ""

        return metadata

    # ── Portada ───────────────────────────────────────────────────────────────

    @staticmethod
    def find_cover(audio_path: str) -> Optional[str]:
        """
        Localiza la portada del audio:
        1. Busca un archivo de imagen junto al audio (mismo nombre base).
        2. Si no existe, intenta extraer la portada incrustada en los tags.
        """
        base_path = os.path.splitext(audio_path)[0]
        for ext in COVER_EXTENSIONS:
            candidate = base_path + ext
            if os.path.isfile(candidate):
                return candidate
        # Fallback: extraer portada incrustada en los tags
        return MetadataExtractor.extract_embedded_cover(audio_path)

    @staticmethod
    def extract_embedded_cover(audio_path: str) -> Optional[str]:
        """
        Extrae la portada incrustada en los tags del audio (ID3 APIC, FLAC pictures,
        MP4 covr) y la guarda como un archivo .webp junto al audio para reutilizarla.
        Devuelve la ruta del archivo generado, o None si no hay portada.
        """
        try:
            from PIL import Image

            audio = MutagenFile(audio_path)
            if audio is None:
                return None

            img_data: Optional[bytes] = None
            tags = getattr(audio, "tags", None)

            # ID3 (MP3) — tag APIC
            if tags and hasattr(tags, "keys"):
                for key in tags.keys():
                    if key.startswith("APIC"):
                        img_data = tags[key].data
                        break

            # FLAC / OGG Vorbis — .pictures
            if img_data is None and hasattr(audio, "pictures") and audio.pictures:
                img_data = audio.pictures[0].data

            # MP4 / M4A — 'covr' tag
            if img_data is None and tags and "covr" in tags:
                covr = tags["covr"]
                if covr:
                    img_data = bytes(covr[0])

            if img_data:
                cache_path = os.path.splitext(audio_path)[0] + ".webp"
                Image.open(io.BytesIO(img_data)).convert("RGB").save(cache_path, "WEBP")
                return cache_path

        except Exception:
            pass

        return None

    # ── Letras (.lrc) ─────────────────────────────────────────────────────────

    @staticmethod
    def get_lrc_path(audio_path: str) -> str:
        """Devuelve la ruta esperada del .lrc para un audio dado."""
        return os.path.splitext(audio_path)[0] + ".lrc"

    @staticmethod
    def load_lyrics(audio_path: str) -> Optional[Dict[float, str]]:
        """
        Carga las letras sincronizadas para un archivo de audio.
        1. Si existe un .lrc en disco, lo parsea y lo devuelve.
        2. Si no existe, devuelve None (la descarga ocurre en background).
        """
        lrc_path = MetadataExtractor.get_lrc_path(audio_path)
        if os.path.isfile(lrc_path):
            return MetadataExtractor.parse_lrc(lrc_path)
        return None

    @staticmethod
    def download_lyrics_sync(
        audio_path: str,
        title: str,
        artist: str,
    ) -> Optional[Dict[float, str]]:
        """
        Descarga letras sincronizadas de forma síncrona.
        Diseñado para usarse con asyncio.run_in_executor() y no bloquear el event loop.
        Devuelve el dict {timestamp: línea} o None si no se encuentran.
        """
        lrc_path = MetadataExtractor.get_lrc_path(audio_path)

        # Si ya existen en disco, devolverlas directamente
        if os.path.isfile(lrc_path):
            return MetadataExtractor.parse_lrc(lrc_path)

        try:
            import syncedlyrics
            content = syncedlyrics.search(f"{title} {artist}", synced_only=True)
            if content:
                with open(lrc_path, "w", encoding="utf-8") as f:
                    f.write(content)
                return MetadataExtractor.parse_lrc(lrc_path)
        except Exception:
            pass

        return None

    @staticmethod
    def parse_lrc(lrc_path: str) -> Dict[float, str]:
        """
        Parsea un archivo .lrc y devuelve un dict {segundos: línea}.
        Soporta formato [mm:ss.xx] y [mm:ss.xxx].
        Ejemplo: {15.5: "I'm a creep", 18.2: "I'm a weirdo"}
        """
        lyrics: Dict[float, str] = {}
        # Regex para timestamps: [01:15.30] o [01:15.300]
        pattern = re.compile(r"\[(\d{1,2}):(\d{2})\.(\d{2,3})\](.*)")

        try:
            with open(lrc_path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    match = pattern.match(line)
                    if match:
                        minutes  = int(match.group(1))
                        seconds  = int(match.group(2))
                        ms_raw   = match.group(3)
                        text     = match.group(4).strip()

                        # Normalizar milisegundos a 2 decimales
                        ms = int(ms_raw) / (1000 if len(ms_raw) == 3 else 100)
                        timestamp = minutes * 60 + seconds + ms

                        if text:  # ignorar líneas vacías
                            lyrics[timestamp] = text
        except Exception:
            pass

        return lyrics

    # ── Helper interno ────────────────────────────────────────────────────────

    @staticmethod
    def _get_tag(tags: Any, key: str, default: str) -> str:
        value = tags.get(key)
        if value is None:
            return default
        if isinstance(value, list):
            return str(value[0]) if value else default
        return str(value)
