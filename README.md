# ShellBeat

Reproductor de musica TUI (Terminal User Interface) escrito en Go con Bubbletea, Lipgloss y mpv.

---

## Historial de Versiones

### v1.0.0 (Version Actual - Migracion a Go)
- Reescritura completa del nucleo en Go para mejor rendimiento, menor consumo de recursos e inicio instantaneo.
- Interfaz de terminal responsiva dividida en 3 paneles balanceados:
  - Panel Izquierdo (30%): Biblioteca de audio y barra de busqueda en tiempo real.
  - Panel Central (33%): Informacion de la pista activa, renderizado de portada en terminal, barra de progreso minimalista, medidor de volumen, modos de reproduccion y cola de reproduccion proxima.
  - Panel Derecho (37%): Letras sincronizadas LRC con desplazamiento automatico y resaltado en tiempo real.
- Visualizacion de portada del album en terminal utilizando renderizado ANSI TrueColor de medios bloques (`▀`) con recorte automatico 1:1 para miniaturas 16:9 de YouTube.
- Telemetria de audio en tiempo real (codec, bitrate en kbps, frecuencia de muestreo en kHz y canales).
- Indicador visual de favoritos en la biblioteca, en el panel principal y en el modal de informacion de pista.
- Barra de progreso minimalista compuesta por una linea fina y un cursor cuadrado deslizante (`■`).
- Motor de reproduccion basado en socket IPC de mpv con soporte para formatos OPUS, FLAC, MP3, OGG, WAV, M4A, AAC y WV.
- Busqueda e integracion de letras sincronizadas con algoritmo de busqueda multi-etapa en la API de LRCLIB (exacta, titulo limpio + artista, titulo limpio).
- Descargador interactivo de playlists en Go (`cmd/downloader`) utilizando `yt-dlp` con registro de descargas para evitar duplicados (`descargadas.txt`).
- Sistema de base de datos local SQLite para persistencia de playlists, favoritos, historial de reproduccion y configuracion.
- Selector de 12 temas de colores de acento.

### v0.1.0 (Version Legacy Python)
- Version inicial construida en Python con Textual y mpv.

---

## Funcionalidades del Reproductor

- Reproduccion de Audio: Control de reproduccion (play, pausa, detener, siguiente, anterior, avance y retroceso de 10 segundos, ajuste de volumen de 0% a 150%).
- Modos de Reproduccion: Modo aleatorio (shuffle) y modos de repeticion (desactivado, repetir todo, repetir una pista).
- Telemetria de Audio: Monitoreo en tiempo real de codec, bitrate, frecuencia de muestreo y numero de canales.
- Portada en Terminal: Renderizado TrueColor ANSI en alta definicion con auto-crop 1:1 para eliminar bordes negros o laterales.
- Barra de Progreso Minimalista: Linea fina con cursor cuadrado (`■`) que indica el avance exacto del audio.
- Indicador de Favoritos: Marca visual en la lista de canciones y badge destacado en la pantalla principal.
- Busqueda e Historial: Busqueda en tiempo real en la biblioteca, historial de pistas reproducidas y marcadores de favoritos.
- Descarga de Playlists: Descargador interactivo en Go que solicita URL o ID de la playlist y evita volver a descargar archivos ya existentes.
- Gestion de Playlists: Creacion y seleccion de listas de reproduccion personalizadas.
- Visualizador de Letras: Sincronizacion de letras LRC por marcas de tiempo en segundos.
- Informacion de Pista: Visualizacion de metadatos (Titulo, Artista, Album, Año, Ruta del archivo, Formato, Telemetria, Estado de Favorito).

---

## Atajos de Teclado

| Tecla | Accion |
|-------|--------|
| space | Play / Pausa |
| n | Siguiente pista |
| p | Pista anterior |
| Izquierda / Derecha | Retroceder / Avanzar 10 segundos |
| + / - | Subir / Bajar volumen 5% |
| s | Alternar modo aleatorio (shuffle) |
| r | Alternar modo de repeticion (off -> all -> one) |
| / | Enfocar barra de busqueda |
| f | Marcar o desmarcar favorito |
| a | Agregar pista a la cola de reproduccion |
| t | Abrir selector de tema de color |
| c | Crear nueva playlist |
| l | Seleccionar playlist |
| i | Mostrar informacion de la pista |
| h | Mostrar historial de reproduccion |
| q / Ctrl+C | Salir de la aplicacion |

---

## Descargador de Playlists (Go)

ShellBeat incluye un descargador de playlists escrito en Go que utiliza `yt-dlp` para guardar el audio en formato **OPUS** de maxima calidad e incrustar la portada y metadatos:

```bash
# Compilar el descargador
go build -o shellbeat-dl ./cmd/downloader

# Ejecutar de forma interactiva
./shellbeat-dl
```

El descargador solicitará la URL o ID de la playlist de YouTube Music (o usará la playlist por defecto al presionar Enter). Utiliza `musica/descargadas.txt` para comparar e ignorar pistas previamente descargadas.

---

## Requisitos e Instalacion

### Requisitos del Sistema
- Go 1.22 o superior
- mpv e yt-dlp instalados en el sistema

### Compilacion y Ejecucion

```bash
# 1. Clonar el repositorio
git clone https://github.com/Edwin16x/ShellBeat.git
cd ShellBeat

# 2. Compilar la aplicacion principal y el descargador
go build -o shellbeat .
go build -o shellbeat-dl ./cmd/downloader

# 3. Descargar musica o colocarla en la carpeta musica/
./shellbeat-dl

# 4. Ejecutar el reproductor
./shellbeat
```

---

## Estructura del Proyecto

```
ShellBeat/
├── main.go               # Punto de entrada principal
├── cmd/
│   └── downloader/       # Descargador interactivo de playlists de YouTube Music en Go
├── go.mod / go.sum       # Modulo Go y dependencias
├── musica/               # Carpeta de biblioteca de audio
└── pkg/
    ├── db/               # Gestor de base de datos SQLite
    ├── metadata/         # Extractor de etiquetas, letras LRC y renderizador de portadas ANSI
    ├── player/           # Controlador mpv via socket IPC y telemetria de audio
    ├── scanner/          # Escaner de directorio de audio
    └── ui/               # Interfaz TUI con Bubbletea y Lipgloss
```

---

## Licencia

MIT
