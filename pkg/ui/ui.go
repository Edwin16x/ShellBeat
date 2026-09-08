package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Edwin16x/ShellBeat/pkg/db"
	"github.com/Edwin16x/ShellBeat/pkg/metadata"
	"github.com/Edwin16x/ShellBeat/pkg/player"
	"github.com/Edwin16x/ShellBeat/pkg/scanner"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var ColorPresets = map[string]string{
	"Morado":    "#7F77DD",
	"Azul":      "#5B8DEF",
	"Verde":     "#4ADE80",
	"Rojo":      "#EF4444",
	"Naranja":   "#F59E0B",
	"Rosa":      "#EC4899",
	"Cyan":      "#22D3EE",
	"Dorado":    "#FBBF24",
	"Lavanda":   "#A78BFA",
	"Esmeralda": "#34D399",
	"Coral":     "#FB7185",
	"Lima":      "#A3E635",
}

var ColorNames = []string{
	"Morado", "Azul", "Verde", "Rojo", "Naranja", "Rosa",
	"Cyan", "Dorado", "Lavanda", "Esmeralda", "Coral", "Lima",
}

type TickMsg time.Time

type Model struct {
	db          *db.DB
	player      *player.Player
	scanner     *scanner.Scanner
	meta        *metadata.MetadataExtractor
	musicFolder string

	allTracks    []string
	tracks       []string
	trackArtists map[string]string
	libraryIndex int

	searchInput textinput.Model
	isSearching bool

	curMeta     metadata.TrackInfo
	lyrics      []metadata.LyricLine
	activeLyric int

	accentColor string

	width  int
	height int

	activeModal    string // "none", "theme", "playlist_select", "playlist_create", "history", "info"
	modalIndex     int
	newPLInput     textinput.Model
	playlists      []db.Playlist
	historyItems   []db.HistoryItem
	statusMessage  string
	statusExpireAt time.Time
}

func NewModel(dbConn *db.DB, plyr *player.Player, musicDir string) Model {
	metaExtractor := metadata.New()
	scn := scanner.New()

	ti := textinput.New()
	ti.Placeholder = "🔍 buscar..."
	ti.CharLimit = 50

	plInput := textinput.New()
	plInput.Placeholder = "Nombre de la playlist..."
	plInput.CharLimit = 40

	accent := dbConn.GetConfig("accent_color", "#7F77DD")

	m := Model{
		db:           dbConn,
		player:       plyr,
		scanner:      scn,
		meta:         metaExtractor,
		musicFolder:  musicDir,
		searchInput:  ti,
		newPLInput:   plInput,
		accentColor:  accent,
		trackArtists: make(map[string]string),
		activeModal:  "none",
	}

	tracks, _ := scn.ScanFolder(musicDir)
	m.allTracks = tracks
	m.tracks = tracks
	m.player.LoadPlaylist(tracks)

	// Preload track metadata
	go m.preloadMetadata(tracks)

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) preloadMetadata(tracks []string) {
	for _, tr := range tracks {
		meta := m.meta.GetMetadata(tr)
		if meta.Artist != "" && meta.Artist != "Artista Desconocido" {
			m.trackArtists[tr] = meta.Artist
		}
	}
}

type LyricsLoadedMsg struct {
	FilePath string
	Lyrics   []metadata.LyricLine
	Err      error
}

func fetchLyricsCmd(meta *metadata.MetadataExtractor, path, title, artist string) tea.Cmd {
	return func() tea.Msg {
		lyrics, err := meta.DownloadLyricsLRCLIB(path, title, artist)
		return LyricsLoadedMsg{
			FilePath: path,
			Lyrics:   lyrics,
			Err:      err,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case TickMsg:
		if m.player.CurrentTrackPath() != "" && m.curMeta.FilePath != m.player.CurrentTrackPath() {
			cmd := m.onTrackChanged(m.player.CurrentTrackPath())
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		m.updateActiveLyric()
		if time.Now().After(m.statusExpireAt) {
			m.statusMessage = ""
		}
		cmds = append(cmds, tickCmd())

	case LyricsLoadedMsg:
		if msg.FilePath == m.curMeta.FilePath {
			if msg.Err == nil && len(msg.Lyrics) > 0 {
				m.lyrics = msg.Lyrics
				m.activeLyric = -1
			}
		}

	case tea.KeyMsg:
		if m.activeModal != "none" {
			return m.updateModal(msg)
		}

		if m.isSearching {
			switch msg.String() {
			case "esc":
				m.isSearching = false
				m.searchInput.Blur()
			case "enter":
				m.isSearching = false
				m.searchInput.Blur()
				if len(m.tracks) > 0 && m.libraryIndex < len(m.tracks) {
					m.player.Play(m.libraryIndex)
					cmd := m.onTrackChanged(m.tracks[m.libraryIndex])
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
				}
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filterTracks()
			}
			return m, tea.Batch(cmds...)
		}

		// Normal keybindings
		switch msg.String() {
		case "q", "ctrl+c":
			m.player.Close()
			return m, tea.Quit

		case "/":
			m.isSearching = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "space":
			m.player.TogglePause()

		case "n":
			m.player.Next()
			if m.player.CurrentTrackPath() != "" {
				cmd := m.onTrackChanged(m.player.CurrentTrackPath())
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "p":
			m.player.Previous()
			if m.player.CurrentTrackPath() != "" {
				cmd := m.onTrackChanged(m.player.CurrentTrackPath())
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "right":
			m.player.Seek(10, true)

		case "left":
			m.player.Seek(-10, true)

		case "+", "=":
			m.player.SetVolume(m.player.Volume() + 5)

		case "-":
			m.player.SetVolume(m.player.Volume() - 5)

		case "s":
			shuf := m.player.ToggleShuffle()
			if shuf {
				m.setStatus("🔀 Shuffle activado")
			} else {
				m.setStatus("➡️ Shuffle desactivado")
			}

		case "r":
			mode := m.player.CycleRepeat()
			m.setStatus(fmt.Sprintf("🔁 Repeat: %s", mode))

		case "f":
			if m.curMeta.FilePath != "" {
				fav, _ := m.db.ToggleFavorite(m.curMeta.FilePath)
				if fav {
					m.setStatus("♥ Añadido a Favoritos")
				} else {
					m.setStatus("♡ Quitado de Favoritos")
				}
			}

		case "t":
			m.activeModal = "theme"
			m.modalIndex = 0

		case "i":
			m.activeModal = "info"

		case "h":
			m.activeModal = "history"
			items, _ := m.db.GetHistory(20)
			m.historyItems = items

		case "c":
			m.activeModal = "playlist_create"
			m.newPLInput.Focus()
			return m, textinput.Blink

		case "l":
			m.activeModal = "playlist_select"
			pls, _ := m.db.GetPlaylists()
			m.playlists = pls
			m.modalIndex = 0

		case "a":
			if len(m.tracks) > 0 && m.libraryIndex < len(m.tracks) {
				m.player.AddToQueue(m.libraryIndex)
				m.setStatus("➕ Añadido a la cola")
			}

		case "up":
			if m.libraryIndex > 0 {
				m.libraryIndex--
			}

		case "down":
			if m.libraryIndex < len(m.tracks)-1 {
				m.libraryIndex++
			}

		case "enter":
			if len(m.tracks) > 0 && m.libraryIndex < len(m.tracks) {
				m.player.Play(m.libraryIndex)
				cmd := m.onTrackChanged(m.tracks[m.libraryIndex])
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) filterTracks() {
	query := strings.ToLower(strings.TrimSpace(m.searchInput.Value()))
	if query == "" {
		m.tracks = m.allTracks
	} else {
		var filtered []string
		for _, tr := range m.allTracks {
			name := strings.ToLower(filepath.Base(tr))
			if strings.Contains(name, query) {
				filtered = append(filtered, tr)
			}
		}
		m.tracks = filtered
	}
	m.libraryIndex = 0
	m.player.LoadPlaylist(m.tracks)
}

func (m *Model) onTrackChanged(path string) tea.Cmd {
	if path == "" {
		return nil
	}

	m.curMeta = m.meta.GetMetadata(path)
	_ = m.db.AddHistory(path)

	lyrics, loaded := m.meta.LoadLyrics(path)
	if loaded {
		m.lyrics = lyrics
		m.activeLyric = -1
		return nil
	}

	m.lyrics = nil
	m.activeLyric = -1

	title := m.curMeta.Title
	artist := m.curMeta.Artist
	return fetchLyricsCmd(m.meta, path, title, artist)
}

func (m *Model) updateActiveLyric() {
	if len(m.lyrics) == 0 {
		return
	}
	pos := m.player.Position()
	active := -1
	for i, l := range m.lyrics {
		if pos >= l.Timestamp {
			active = i
		} else {
			break
		}
	}
	m.activeLyric = active
}

func (m *Model) setStatus(msg string) {
	m.statusMessage = msg
	m.statusExpireAt = time.Now().Add(2 * time.Second)
}

func (m Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.activeModal = "none"
		m.newPLInput.Blur()
		return m, nil
	}

	switch m.activeModal {
	case "theme":
		switch msg.String() {
		case "up":
			if m.modalIndex > 0 {
				m.modalIndex--
			}
		case "down":
			if m.modalIndex < len(ColorNames)-1 {
				m.modalIndex++
			}
		case "enter":
			colorName := ColorNames[m.modalIndex]
			m.accentColor = ColorPresets[colorName]
			_ = m.db.SetConfig("accent_color", m.accentColor)
			m.activeModal = "none"
			m.setStatus("🎨 Tema actualizado: " + colorName)
		}

	case "playlist_create":
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.newPLInput.Value())
			if name != "" {
				_, err := m.db.CreatePlaylist(name)
				if err == nil {
					m.setStatus("📁 Playlist creada: " + name)
				}
			}
			m.newPLInput.Reset()
			m.newPLInput.Blur()
			m.activeModal = "none"
		default:
			var cmd tea.Cmd
			m.newPLInput, cmd = m.newPLInput.Update(msg)
			return m, cmd
		}

	case "playlist_select":
		totalOptions := len(m.playlists) + 1
		switch msg.String() {
		case "up":
			if m.modalIndex > 0 {
				m.modalIndex--
			}
		case "down":
			if m.modalIndex < totalOptions-1 {
				m.modalIndex++
			}
		case "enter":
			if m.modalIndex == 0 {
				m.tracks = m.allTracks
				m.player.LoadPlaylist(m.allTracks)
				m.setStatus("🎵 Todas las pistas cargadas")
			} else {
				pl := m.playlists[m.modalIndex-1]
				plTracks, err := m.db.GetPlaylistTracks(pl.ID)
				if err == nil && len(plTracks) > 0 {
					m.tracks = plTracks
					m.player.LoadPlaylist(plTracks)
					m.setStatus("📁 Cargada playlist: " + pl.Name)
				}
			}
			m.activeModal = "none"
		}

	case "history", "info":
		if msg.String() == "enter" {
			m.activeModal = "none"
		}
	}

	return m, nil
}

// ── View Rendering ────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Inicializando ShellBeat..."
	}

	accent := lipgloss.Color(m.accentColor)

	// Base Layout Dimensions
	mainHeight := m.height - 4
	if mainHeight < 10 {
		mainHeight = 10
	}

	sideWidth := m.width / 4
	if sideWidth < 25 {
		sideWidth = 25
	}
	lyricsWidth := m.width / 4
	if lyricsWidth < 25 {
		lyricsWidth = 25
	}
	playerWidth := m.width - sideWidth - lyricsWidth - 6
	if playerWidth < 30 {
		playerWidth = 30
	}

	// 1. Sidebar Panel (Library & Search)
	sidebar := m.renderSidebar(sideWidth, mainHeight, accent)

	// 2. Player Panel (Now Playing, Progress Bar, Controls, Queue)
	playerView := m.renderPlayerView(playerWidth, mainHeight, accent)

	// 3. Lyrics Panel
	lyricsView := m.renderLyricsView(lyricsWidth, mainHeight, accent)

	mainRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, playerView, lyricsView)

	// Footer
	footer := m.renderFooter(accent)

	fullView := lipgloss.JoinVertical(lipgloss.Left, mainRow, footer)

	if m.activeModal != "none" {
		return m.renderModalOverlay(fullView, accent)
	}

	return fullView
}

func (m Model) renderSidebar(width, height int, accent lipgloss.Color) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Padding(0, 1)

	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	header := headerStyle.Render("  BIBLIOTECA ") + badgeStyle.Render(fmt.Sprintf("[%d pistas]", len(m.tracks)))

	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Width(width - 4).
		Render(m.searchInput.View())

	listHeight := height - 6
	if listHeight < 3 {
		listHeight = 3
	}

	var listLines []string
	startIdx := 0
	if m.libraryIndex > listHeight-1 {
		startIdx = m.libraryIndex - (listHeight - 1)
	}

	endIdx := startIdx + listHeight
	if endIdx > len(m.tracks) {
		endIdx = len(m.tracks)
	}

	for i := startIdx; i < endIdx; i++ {
		tr := m.tracks[i]
		clean := metadata.FormatCleanTitle(filepath.Base(tr))
		if len(clean) > width-8 {
			clean = clean[:width-10] + "..."
		}

		isFav := m.db.IsFavorite(tr)
		prefix := "  "
		if isFav {
			prefix = "♥ "
		}

		line := fmt.Sprintf("%s%s", prefix, clean)
		if i == m.libraryIndex {
			lineStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(accent)
			listLines = append(listLines, lineStyle.Render(fmt.Sprintf("> %-*s", width-6, line)))
		} else {
			lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
			listLines = append(listLines, lineStyle.Render(fmt.Sprintf("  %-*s", width-6, line)))
		}
	}

	listContent := strings.Join(listLines, "\n")

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Width(width).
		Height(height)

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, searchBox, listContent))
}

func (m Model) renderPlayerView(width, height int, accent lipgloss.Color) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Padding(0, 1)

	header := headerStyle.Render("  AHORA SUENA")

	trackTitle := "Sin canción seleccionada"
	artistAlbum := "—"
	ext := "OPUS"
	isFav := false

	if m.curMeta.FilePath != "" {
		trackTitle = m.curMeta.Title
		artistAlbum = fmt.Sprintf("%s • %s", m.curMeta.Artist, m.curMeta.Album)
		ext = strings.ToUpper(strings.TrimPrefix(filepath.Ext(m.curMeta.FilePath), "."))
		isFav = m.db.IsFavorite(m.curMeta.FilePath)
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		MarginTop(1).
		PaddingLeft(2)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		PaddingLeft(2)

	badgeStyle := lipgloss.NewStyle().
		Foreground(accent).
		Border(lipgloss.NormalBorder()).
		BorderForeground(accent).
		Padding(0, 1).
		MarginLeft(2)

	favBadgeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#EF4444")).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#EF4444")).
		Padding(0, 1).
		MarginLeft(1)

	badges := badgeStyle.Render(ext)
	if isFav {
		badges += favBadgeStyle.Render("♥ FAVORITO")
	}

	// Audio Telemetry
	telemetryStr := m.player.TelemetryString()
	telemetryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		PaddingLeft(2).
		MarginTop(0)

	// Progress bar calculation
	pos := m.player.Position()
	dur := m.player.Duration()
	pct := 0.0
	if dur > 0 {
		pct = pos / dur
	}
	if pct > 1.0 {
		pct = 1.0
	}

	barWidth := width - 12
	if barWidth < 10 {
		barWidth = 10
	}
	filledLen := int(pct * float64(barWidth))
	if filledLen < 0 {
		filledLen = 0
	}
	emptyLen := barWidth - filledLen

	barFilled := strings.Repeat("█", filledLen)
	barEmpty := strings.Repeat("░", emptyLen)

	barStyle := lipgloss.NewStyle().Foreground(accent)
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	progressBarStr := barStyle.Render(barFilled) + emptyStyle.Render(barEmpty)

	timeStr := fmt.Sprintf("%s / %s", formatSeconds(pos), formatSeconds(dur))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).PaddingLeft(2)

	// Volume & Mode line
	vol := m.player.Volume()
	volBarWidth := 10
	volFilled := int(float64(vol) / 150.0 * float64(volBarWidth))
	if volFilled > volBarWidth {
		volFilled = volBarWidth
	}
	volStr := fmt.Sprintf("vol [%s%s] %d%%", strings.Repeat("▰", volFilled), strings.Repeat("▱", volBarWidth-volFilled), vol)

	shufStr := "🔀"
	if !m.player.IsShuffle() {
		shufStr = "➡️"
	}
	repStr := fmt.Sprintf("🔁 %s", m.player.RepeatMode())
	modeLine := fmt.Sprintf("%s  |  %s  |  %s", volStr, shufStr, repStr)
	modeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).PaddingLeft(2).MarginTop(1)

	// Queue preview
	queueHeader := lipgloss.NewStyle().Bold(true).Foreground(accent).PaddingLeft(2).MarginTop(1).Render("Siguiente en la cola:")
	upcomingIdxs := m.player.GetUpcoming(4)
	var queueLines []string
	for i, idx := range upcomingIdxs {
		if idx >= 0 && idx < len(m.tracks) {
			tName := metadata.FormatCleanTitle(filepath.Base(m.tracks[idx]))
			queueLines = append(queueLines, fmt.Sprintf("  %d. %s", i+1, tName))
		}
	}
	queueContent := strings.Join(queueLines, "\n")
	if queueContent == "" {
		queueContent = "  (cola vacía)"
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		badges,
		titleStyle.Render(trackTitle),
		subtitleStyle.Render(artistAlbum),
		telemetryStyle.Render("📊 "+telemetryStr),
		lipgloss.NewStyle().PaddingLeft(2).MarginTop(1).Render(progressBarStr),
		timeStyle.Render(timeStr),
		modeStyle.Render(modeLine),
		queueHeader,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(queueContent),
	)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Width(width).
		Height(height)

	return panelStyle.Render(content)
}

func (m Model) renderLyricsView(width, height int, accent lipgloss.Color) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Padding(0, 1)

	header := headerStyle.Render("  LETRA ")

	var lyricsLines []string
	if len(m.lyrics) == 0 {
		lyricsLines = append(lyricsLines, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Padding(1, 2).Render("Sin letra sincronizada"))
	} else {
		maxLines := height - 4
		start := 0
		if m.activeLyric > maxLines/2 {
			start = m.activeLyric - maxLines/2
		}
		end := start + maxLines
		if end > len(m.lyrics) {
			end = len(m.lyrics)
		}

		for i := start; i < end; i++ {
			line := m.lyrics[i].Text
			if len(line) > width-6 {
				line = line[:width-8] + "..."
			}

			if i == m.activeLyric {
				lyrStyle := lipgloss.NewStyle().Bold(true).Foreground(accent)
				lyricsLines = append(lyricsLines, lyrStyle.Render("▶ "+line))
			} else {
				dist := i - m.activeLyric
				if dist < 0 {
					dist = -dist
				}
				color := "#777777"
				if dist > 3 {
					color = "#444444"
				}
				lyrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
				lyricsLines = append(lyricsLines, lyrStyle.Render("  "+line))
			}
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(lyricsLines, "\n"))

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Width(width).
		Height(height)

	return panelStyle.Render(content)
}

func (m Model) renderFooter(accent lipgloss.Color) string {
	helpStr := "[space] Play/Pausa | [n/p] Sig/Ant | [←/→] Seek | [+/-] Vol | [s] Shuffle | [r] Repeat | [f] Fav | [t] Tema | [c/l] Playlist | [i] Info | [h] Historial | [/] Buscar | [q] Salir"

	statusStr := ""
	if m.statusMessage != "" {
		statusStr = lipgloss.NewStyle().Bold(true).Foreground(accent).Render("  " + m.statusMessage)
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1)

	return lipgloss.JoinHorizontal(lipgloss.Left, footerStyle.Render(helpStr), statusStr)
}

func (m Model) renderModalOverlay(baseView string, accent lipgloss.Color) string {
	modalWidth := 46
	var dialogContent string

	switch m.activeModal {
	case "theme":
		var lines []string
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Seleccionar Color de Acento:"))
		for i, name := range ColorNames {
			colorHex := ColorPresets[name]
			sample := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex)).Render("■■■ ")
			line := fmt.Sprintf("  %s %s", sample, name)
			if i == m.modalIndex {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accent).Render("> " + name)
			}
			lines = append(lines, line)
		}
		dialogContent = strings.Join(lines, "\n")

	case "playlist_create":
		title := lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Crear Nueva Playlist:")
		dialogContent = lipgloss.JoinVertical(lipgloss.Left, title, m.newPLInput.View(), "\n[Enter] Guardar  |  [Esc] Cancelar")

	case "playlist_select":
		var lines []string
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Seleccionar Playlist:"))

		allOption := "  Todas las pistas"
		if m.modalIndex == 0 {
			allOption = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accent).Render("> Todas las pistas")
		}
		lines = append(lines, allOption)

		for i, pl := range m.playlists {
			line := fmt.Sprintf("  %s", pl.Name)
			if i+1 == m.modalIndex {
				line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(accent).Render("> " + pl.Name)
			}
			lines = append(lines, line)
		}
		dialogContent = strings.Join(lines, "\n")

	case "history":
		var lines []string
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Historial de Reproducción:"))
		if len(m.historyItems) == 0 {
			lines = append(lines, "  (historial vacío)")
		} else {
			for _, item := range m.historyItems {
				tName := metadata.FormatCleanTitle(filepath.Base(item.TrackPath))
				lines = append(lines, fmt.Sprintf("  • %s", tName))
			}
		}
		lines = append(lines, "\n[Enter / Esc] Cerrar")
		dialogContent = strings.Join(lines, "\n")

	case "info":
		var lines []string
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(accent).Render("Información de la Pista:"))
		lines = append(lines, fmt.Sprintf("  Título:      %s", m.curMeta.Title))
		lines = append(lines, fmt.Sprintf("  Artista:     %s", m.curMeta.Artist))
		lines = append(lines, fmt.Sprintf("  Álbum:       %s", m.curMeta.Album))
		lines = append(lines, fmt.Sprintf("  Año:         %s", m.curMeta.Year))

		favStatus := "♡ No"
		if m.db.IsFavorite(m.curMeta.FilePath) {
			favStatus = "♥ Sí (Favorito)"
		}
		lines = append(lines, fmt.Sprintf("  Favorito:    %s", favStatus))

		telemetryStr := m.player.TelemetryString()
		lines = append(lines, fmt.Sprintf("  Telemetría:  %s", telemetryStr))
		lines = append(lines, fmt.Sprintf("  Ruta:        %s", m.curMeta.FilePath))
		lines = append(lines, "\n[Enter / Esc] Cerrar")
		dialogContent = strings.Join(lines, "\n")
	}

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accent).
		Padding(1, 2).
		Width(modalWidth).
		Render(dialogContent)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialogBox)
}

func formatSeconds(sec float64) string {
	s := int(sec)
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
