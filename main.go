package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Edwin16x/ShellBeat/pkg/db"
	"github.com/Edwin16x/ShellBeat/pkg/player"
	"github.com/Edwin16x/ShellBeat/pkg/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Locate music folder
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	musicDir := filepath.Join(cwd, "musica")
	_ = os.MkdirAll(musicDir, 0755)

	// Initialize Database
	dbConn, err := db.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al inicializar base de datos: %v\n", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	// Initialize Player
	plyr, err := player.NewPlayer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al inicializar reproductor (asegúrate de tener 'mpv' instalado): %v\n", err)
		os.Exit(1)
	}
	defer plyr.Close()

	// Initialize UI Model
	model := ui.NewModel(dbConn, plyr, musicDir)

	// Launch Bubbletea Program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error ejecutando ShellBeat: %v\n", err)
		os.Exit(1)
	}
}
