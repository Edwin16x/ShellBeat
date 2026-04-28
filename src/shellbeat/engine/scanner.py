import os
from typing import List

# Importamos las extensiones desde metadata para mantener la consistencia
from .metadata import AUDIO_EXTENSIONS

class LibraryScanner:
    """
    Se encarga de localizar archivos de audio en el sistema de archivos.
    """

    @staticmethod
    def scan_folder(folder_path: str) -> List[str]:
        """
        Escanea una carpeta de forma recursiva y devuelve una lista 
        de rutas absolutas a archivos de audio compatibles.
        """
        found_tracks: List[str] = []
        
        # Validar que la ruta existe
        if not os.path.exists(folder_path):
            return []

        for root, _, files in os.walk(folder_path):
            # Ordenamos los archivos para que la playlist sea predecible
            for f in sorted(files):
                extension = os.path.splitext(f)[1].lower()
                if extension in AUDIO_EXTENSIONS:
                    full_path = os.path.join(root, f)
                    found_tracks.append(os.path.normpath(full_path))
        
        return found_tracks

    @staticmethod
    def is_valid_track(path: str) -> bool:
        """Verifica si un archivo individual es un track soportado."""
        return (
            os.path.isfile(path) and 
            os.path.splitext(path)[1].lower() in AUDIO_EXTENSIONS
        )
