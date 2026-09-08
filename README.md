# ShellBeat

Reproductor de musica TUI (Terminal User Interface) escrito en Go con Bubbletea, Lipgloss y mpv.

---

## Historial de Versiones

### v1.0.0 (Version Actual - Migracion a Go)
- Reescritura completa del nucleo en Go para mejor rendimiento, menor consumo de recursos e inicio instantaneo.
- Interfaz de terminal responsiva dividida en 3 paneles:
  - Panel Izquierdo: Biblioteca de audio y barra de busqueda en tiempo real.
  - Panel Central: Informacion de la pista activa, barra de progreso, medidor de volumen, modos de reproduccion y cola de reproduccion proxima.
  - Panel Derecho: Letras sincronizadas LRC con desplazamiento automatico y resaltado en tiempo real.
- Motor de reproduccion basado en socket IPC de mpv con soporte para formatos OPUS, FLAC, MP3, OGG, WAV, M4A, AAC y WV.
- Descarga automatica de letras sincronizadas desde la API de LRCLIB si no se encuentra archivo LRC local.
- Sistema de base de datos local SQLite para persistencia de playlists, favoritos, historial de reproduccion y configuracion.
- Selector de 12 temas de colores de acento.

### v0.1.0 (Version Legacy Python)
- Version inicial construida en Python con Textual y mpv.

---

## Funcionalidades del Reproductor

- Reproduccion de Audio: Control de reproduccion (play, pausa, detener, siguiente, anterior, avance y retroceso de 10 segundos, ajuste de volumen de 0% a 150%).
- Modos de Reproduccion: Modo aleatorio (shuffle) y modos de repeticion (desactivado, repetir todo, repetir una pista).
- Busqueda e Historial: Busqueda en tiempo real en la biblioteca, historial de pistas reproducidas y marcadores de favoritos.
- Gestion de Playlists: Creacion y seleccion de listas de reproduccion personalizadas.
- Visualizador de Letras: Sincronizacion de letras LRC por marcas de tiempo en segundos.
- Informacion de Pista: Visualizacion de metadatos (Titulo, Artista, Album, Año, Ruta del archivo, Formato).

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

## Requisitos e Instalacion

### Requisitos del Sistema
- Go 1.22 o superior
- mpv instalado en el sistema

### Compilacion y Ejecucion

```bash
# 1. Clonar el repositorio
git clone https://github.com/Edwin16x/ShellBeat.git
cd ShellBeat

# 2. Compilar la aplicacion
go build -o shellbeat .

# 3. Crear directorio de musica y agregar archivos
mkdir -p musica

# 4. Ejecutar
./shellbeat
```

---

## Estructura del Proyecto

```
ShellBeat/
├── main.go               # Punto de entrada principal
├── go.mod / go.sum       # Modulo Go y dependencias
├── musica/               # Carpeta de biblioteca de audio
└── pkg/
    ├── db/               # Gestor de base de datos SQLite
    ├── metadata/         # Extractor de etiquetas y letras LRC
    ├── player/           # Controlador mpv via socket IPC
    ├── scanner/          # Escaner de directorio de audio
    └── ui/               # Interfaz TUI con Bubbletea y Lipgloss
```

---

## Licencia

MIT
