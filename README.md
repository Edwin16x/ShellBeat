# 🎵 ShellBeat

> Reproductor de música TUI (Terminal User Interface) construido con Python, Textual y mpv.

ShellBeat es un reproductor de música para la terminal con letras sincronizadas, playlists, temas personalizables y una interfaz elegante de tres paneles. Diseñado para usuarios que viven en la terminal.

---

## ✨ Características

### 🎧 Reproducción
- Reproducción de audio via **libmpv** (formatos: opus, mp3, flac, ogg, m4a, wav)
- **Auto-avance** — las pistas avanzan automáticamente al terminar
- **Shuffle** — modo aleatorio con orden pre-generado
- **Repeat** — tres modos: off → all (🔁) → one (🔂)
- **Cola de reproducción** — agrega pistas para "reproducir después"
- Control de **volumen** (0–150%) con barra visual `▰▱`
- **Seek** ±10 segundos

### 🎤 Letras Sincronizadas
- Descarga automática de letras sincronizadas via `syncedlyrics`
- Visualización estilo **Apple Music** con gradiente de opacidad:
  - Línea activa: color de acento + negrita
  - ±1 línea: 45% opacidad
  - ±2 líneas: 25% opacidad
  - ±3 líneas: 12% opacidad
  - Resto: 6% opacidad
- Scroll automático centrado en la línea activa
- Búsqueda binaria O(log n) para sincronización eficiente

### 📚 Biblioteca y Playlists
- Escaneo automático de la carpeta `musica/`
- **Búsqueda en tiempo real** (`/`) — filtra por título o artista
- Creación de **playlists** desde la app (`c`) — se guardan en SQLite
- Selección de playlists (`l`) para cambiar entre colecciones
- **Favoritos** (`f`) con indicador ♥

### 🎨 Personalización
- **12 temas de color** seleccionables con `t`:
  - Morado, Azul, Verde, Rojo, Naranja, Rosa, Cyan, Dorado, Lavanda, Esmeralda, Coral, Lima
- El color de acento se aplica a: título, barra de progreso, volumen, bordes, badges, letras activas
- **Preferencias persistentes** — se guardan en SQLite y se restauran al reiniciar
- El volumen se guarda al salir

### 📊 Datos e Historial
- **Historial de reproducción** automático (`h`) — registra cada pista reproducida
- **Info detallada** de la pista (`i`) — título, artista, álbum, año, bitrate, path
- Base de datos SQLite en `~/.shellbeat/shellbeat.db`

---

## 📐 Layout

```
┌─────────────┬──────────────────┬───────────────────┐
│  biblioteca │   ahora suena    │      letra        │
│  142 pistas │   opus · 192k    │   sincronizada    │
├─────────────┤──────────────────┤───────────────────┤
│ 🔍 buscar…  │                  │                   │
│             │  ● Título        │  línea anterior   │
│ ● Canción 1 │    Artista       │  línea anterior   │
│   Canción 2 │                  │ ▶ LÍNEA ACTIVA    │
│   Canción 3 │  ████░░░░ 1:38   │  línea siguiente  │
│   Canción 4 │  1:38 ───── 4:19 │  línea siguiente  │
│   Canción 5 │  vol ▰▰▰▰▱▱ 70% │                   │
│   Canción 6 │  🔀 shuffle      │                   │
│             ├──────────────────┤                   │
│             │   siguiente      │                   │
│             │  1. Canción 3    │                   │
│             │  2. Canción 7    │                   │
│             │  3. Canción 1    │                   │
├─────────────┴──────────────────┴───────────────────┤
│ space play │ ←→ seek │ +- vol │ s shuffle │ q quit │
└────────────────────────────────────────────────────┘
```

---

## ⌨️ Atajos de Teclado

| Tecla | Acción |
|-------|--------|
| `space` | Play / Pausa |
| `n` | Siguiente pista |
| `p` | Pista anterior |
| `← →` | Seek ±10 segundos |
| `+ −` | Volumen ±5% |
| `s` | Toggle shuffle 🔀 |
| `r` | Cycle repeat: off → all 🔁 → one 🔂 |
| `/` | Buscar en la biblioteca |
| `f` | Marcar/desmarcar favorito ♥ |
| `a` | Agregar pista a la cola |
| `t` | Cambiar tema de color |
| `c` | Crear nueva playlist |
| `l` | Seleccionar playlist |
| `i` | Info detallada de la pista |
| `h` | Ver historial de reproducción |
| `q` | Salir |

---

## 🚀 Instalación

### Requisitos del sistema
- Python 3.10+
- **mpv** (libmpv) instalado en el sistema
- Terminal con soporte de color verdadero (Kitty, Alacritty, WezTerm, etc.)

### Pasos

```bash
# 1. Clonar el repositorio
git clone https://github.com/tu-usuario/shellbeat.git
cd shellbeat

# 2. Crear entorno virtual
python -m venv venv
source venv/bin/activate

# 3. Instalar dependencias
pip install -r requirements.txt

# 4. Instalar mpv (si no está instalado)
# Arch/Manjaro:
sudo pacman -S mpv

# Ubuntu/Debian:
sudo apt install mpv

# Fedora:
sudo dnf install mpv

# 5. Agregar música a la carpeta musica/
# (o usar el downloader — ver abajo)

# 6. Ejecutar
python run.py
```

---

## 📥 Descargador de Música

ShellBeat incluye un descargador basado en `yt-dlp` que descarga playlists completas de YouTube Music:

```bash
# Descargar la playlist por defecto
python downloader.py

# Descargar una playlist específica
python downloader.py "https://music.youtube.com/playlist?list=PLxxxxxxx"
```

### ¿Qué hace?
- Descarga en formato **opus** (máxima calidad)
- Guarda portadas en **webp**
- Incrusta **metadata** automáticamente
- Usa `descargadas.txt` para **no re-descargar** pistas existentes
- Sleep entre descargas para evitar rate limiting

---

## 🏗️ Arquitectura

```
Reproductor/
├── run.py                  # Punto de entrada
├── downloader.py           # Descargador de playlists (yt-dlp)
├── requirements.txt        # Dependencias Python
├── musica/                 # Carpeta de música (auto-escaneada)
│   ├── *.opus              # Archivos de audio
│   ├── *.webp              # Portadas de álbum
│   └── descargadas.txt     # Registro de descargas
└── src/shellbeat/
    ├── __init__.py
    ├── app.py              # Aplicación principal (Textual)
    ├── style.css           # Estilos de la UI
    ├── color_engine.py     # Extracción de colores dominantes
    ├── cover_widget.py     # Widget de portada (block chars)
    ├── kitty_cover.py      # Protocolo Kitty (legacy)
    └── engine/
        ├── __init__.py
        ├── player.py       # Motor de reproducción (mpv)
        ├── metadata.py     # Extracción de metadata y letras
        ├── scanner.py      # Escáner de biblioteca
        └── db.py           # Base de datos SQLite
```

### Componentes clave

| Módulo | Responsabilidad |
|--------|----------------|
| `player.py` | Controla mpv: play, pause, seek, volume, shuffle, repeat, cola |
| `metadata.py` | Extrae tags (mutagen), descarga letras sincronizadas |
| `scanner.py` | Escanea `musica/` buscando archivos de audio soportados |
| `db.py` | SQLite para config, playlists, favoritos e historial |
| `app.py` | UI con Textual: layout, eventos, modales, sincronización de letras |

---

## 🗄️ Base de Datos

ShellBeat almacena datos en `~/.shellbeat/shellbeat.db` con estas tablas:

| Tabla | Propósito |
|-------|-----------|
| `config` | Preferencias (color de acento, volumen, fondo) |
| `playlists` | Playlists creadas por el usuario |
| `playlist_tracks` | Pistas asociadas a cada playlist |
| `favorites` | Pistas marcadas como favoritas |
| `play_history` | Registro automático de cada reproducción |

---

## 🛠️ Stack Tecnológico

| Tecnología | Uso |
|-----------|-----|
| [Textual](https://textual.textualize.io/) | Framework TUI (interfaz de terminal) |
| [python-mpv](https://github.com/jaseg/python-mpv) | Bindings de libmpv para reproducción |
| [mutagen](https://mutagen.readthedocs.io/) | Lectura de metadata de audio (ID3, Vorbis, etc.) |
| [Pillow](https://pillow.readthedocs.io/) | Procesamiento de imágenes (extracción de colores) |
| [syncedlyrics](https://github.com/rtcq/syncedlyrics) | Descarga de letras sincronizadas (LRC) |
| [yt-dlp](https://github.com/yt-dlp/yt-dlp) | Descarga de audio desde YouTube Music |
| SQLite | Almacenamiento local de preferencias y datos |

---

## 📝 Roadmap

- [ ] Barra de progreso interactiva (click para saltar)
- [ ] Ecualizador visual (ASCII spectrum)
- [ ] Soporte para múltiples carpetas de música
- [ ] Importar/exportar playlists (JSON, M3U)
- [ ] Estadísticas de reproducción (top artistas, canciones más escuchadas)
- [ ] Integración con Last.fm / ListenBrainz (scrobbling)
- [ ] Migración a `pyproject.toml` para instalación via `pip install -e .`
- [ ] Soporte para temas custom (archivo de configuración TOML/YAML)
- [ ] Visualización de portada mejorada (protocolo Sixel/Kitty con fallback)

---

## 📄 Licencia

MIT


