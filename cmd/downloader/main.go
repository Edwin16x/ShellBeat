package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const DefaultPlaylistURL = "https://music.youtube.com/playlist?list=PL8UeZt7lYRzRfVTTYIlPfJEsr__m4fEmy"

func findYtDlp() string {
	// Check user local bin first (updated version)
	home, err := os.UserHomeDir()
	if err == nil {
		userLocal := filepath.Join(home, ".local", "bin", "yt-dlp")
		if _, err := os.Stat(userLocal); err == nil {
			return userLocal
		}
	}

	// Check system PATH
	if path, err := exec.LookPath("yt-dlp"); err == nil {
		return path
	}

	return ""
}

func getExtendedEnv() []string {
	env := os.Environ()
	home, err := os.UserHomeDir()
	if err != nil {
		return env
	}

	// Include node & local bin in PATH if present
	extraPaths := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".nvm", "versions", "node", "v22.20.0", "bin"),
	}

	var currentPath string
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			currentPath = strings.TrimPrefix(e, "PATH=")
			newPath := strings.Join(extraPaths, ":") + ":" + currentPath
			env[i] = "PATH=" + newPath
			break
		}
	}
	return env
}

func main() {
	fmt.Println("==================================================")
	fmt.Println("           ShellBeat Music Downloader             ")
	fmt.Println("==================================================")

	ytDlpPath := findYtDlp()
	if ytDlpPath == "" {
		fmt.Println("Error: 'yt-dlp' no se encuentra instalado en el sistema.")
		fmt.Println("Instálalo con: pip install --break-system-packages -U yt-dlp")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nIngresa la URL o ID de la playlist de YouTube Music:")
	fmt.Printf("[Por defecto: %s]\n> ", DefaultPlaylistURL)

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error leyendo la entrada.")
		os.Exit(1)
	}

	input = strings.TrimSpace(input)
	playlistURL := DefaultPlaylistURL
	if input != "" {
		if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
			playlistURL = input
		} else if strings.HasPrefix(input, "PL") || strings.HasPrefix(input, "OLAK5ui_") {
			playlistURL = "https://music.youtube.com/playlist?list=" + input
		} else {
			playlistURL = input
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	outputFolder := filepath.Join(cwd, "musica")
	_ = os.MkdirAll(outputFolder, 0755)

	archiveFile := filepath.Join(outputFolder, "descargadas.txt")
	outputTemplate := filepath.Join(outputFolder, "%(title)s [%(id)s].%(ext)s")

	fmt.Printf("\n▶ Ejecutando yt-dlp (%s):\n  URL:                %s\n  Destino:            %s/\n  Control duplicados: %s\n\n",
		ytDlpPath, playlistURL, outputFolder, archiveFile)

	args := []string{
		"--format", "bestaudio/best",
		"--extract-audio",
		"--audio-format", "opus",
		"--audio-quality", "0",
		"--write-thumbnail",
		"--convert-thumbnails", "webp",
		"--embed-metadata",
		"--ignore-errors",
		"--no-warnings",
		"--download-archive", archiveFile,
		"--sleep-interval", "1",
		"--max-sleep-interval", "3",
		"--output", outputTemplate,
		playlistURL,
	}

	cmd := exec.Command(ytDlpPath, args...)
	cmd.Env = getExtendedEnv()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("\nProceso de descarga finalizado: %v\n", err)
	} else {
		fmt.Println("\n✔ Descarga completada exitosamente.")
	}
}
