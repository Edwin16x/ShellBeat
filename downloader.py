"""
downloader.py — Descarga una playlist de YouTube Music directamente con yt-dlp.
Uso: python downloader.py [URL_PLAYLIST]
"""
import subprocess
import os
import sys


PLAYLIST_URL = "https://music.youtube.com/playlist?list=PL8UeZt7lYRzRfVTTYIlPfJEsr__m4fEmy"
OUTPUT_FOLDER = "musica"


def download_playlist(url: str, output_folder: str = OUTPUT_FOLDER) -> None:
    os.makedirs(output_folder, exist_ok=True)

    archive  = os.path.join(output_folder, "descargadas.txt")
    template = os.path.join(output_folder, "%(title)s [%(id)s].%(ext)s")

    cmd = [
        "yt-dlp",
        # Formato de audio
        "--format",          "bestaudio",
        "--extract-audio",
        "--audio-format",    "opus",
        "--audio-quality",   "0",
        # Metadata y portada
        "--write-thumbnail",
        "--convert-thumbnails", "webp",
        "--embed-metadata",
        # Comportamiento
        "--ignore-errors",
        "--no-warnings",
        "--download-archive", archive,
        "--sleep-interval",  "1",
        "--max-sleep-interval", "3",
        "--output",          template,
        # URL de la playlist
        url,
    ]

    print(f"▶ Descargando playlist:")
    print(f"  {url}")
    print(f"  → {output_folder}/\n")

    try:
        subprocess.run(cmd, check=False)
    except KeyboardInterrupt:
        print("\n⏹ Descarga interrumpida.")
    except FileNotFoundError:
        print("✗ yt-dlp no encontrado. Instálalo con: pip install yt-dlp")


if __name__ == "__main__":
    url = sys.argv[1] if len(sys.argv) > 1 else PLAYLIST_URL
    download_playlist(url)
