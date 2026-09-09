package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Player struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	sockPath     string
	conn         net.Conn
	playlist     []string
	currentIndex int
	shuffle      bool
	shuffleOrder []int
	shufflePos   int
	repeatMode   string // "off" | "one" | "all"
	manualQueue  []int

	position float64
	duration float64
	volume   int
	isPaused bool
	stopped  bool

	// Telemetry
	audioCodec      string
	audioBitrate    int
	audioSampleRate int
	audioChannels   int

	OnTrackEnd func()
	OnStatus   func()
}

func NewPlayer() (*Player, error) {
	sockPath := fmt.Sprintf("/tmp/shellbeat-mpv-%d.sock", os.Getpid())
	_ = os.Remove(sockPath)

	mpvCmd := exec.Command("mpv",
		"--idle=yes",
		"--video=no",
		"--keep-open=yes",
		fmt.Sprintf("--input-ipc-server=%s", sockPath),
	)

	if err := mpvCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start mpv process: %w", err)
	}

	// Wait for socket creation
	var conn net.Conn
	var err error
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		conn, err = net.Dial("unix", sockPath)
		if err == nil {
			break
		}
	}

	if err != nil {
		_ = mpvCmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to mpv socket: %w", err)
	}

	p := &Player{
		cmd:          mpvCmd,
		sockPath:     sockPath,
		conn:         conn,
		repeatMode:   "all",
		volume:       100,
		currentIndex: -1,
	}

	p.OnTrackEnd = func() {
		p.Next()
	}

	// Start reading IPC responses & observe properties
	go p.listenIPC()
	p.observeProperties()

	return p, nil
}

func (p *Player) observeProperties() {
	_ = p.sendIPC("observe_property", 1, "time-pos")
	_ = p.sendIPC("observe_property", 2, "duration")
	_ = p.sendIPC("observe_property", 3, "pause")
	_ = p.sendIPC("observe_property", 4, "eof-reached")
	_ = p.sendIPC("observe_property", 5, "audio-codec-name")
	_ = p.sendIPC("observe_property", 6, "audio-bitrate")
	_ = p.sendIPC("observe_property", 7, "audio-params/samplerate")
	_ = p.sendIPC("observe_property", 8, "audio-params/channel-count")
}

func (p *Player) sendIPC(cmd string, args ...interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("ipc connection closed")
	}

	fullCmd := append([]interface{}{cmd}, args...)
	req := map[string]interface{}{
		"command": fullCmd,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = p.conn.Write(data)
	return err
}

func (p *Player) listenIPC() {
	scanner := bufio.NewScanner(p.conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var msg struct {
			Event string      `json:"event"`
			Name  string      `json:"name"`
			Data  interface{} `json:"data"`
		}
		if err := json.Unmarshal(line, &msg); err == nil {
			if msg.Event == "property-change" {
				p.handlePropertyChange(msg.Name, msg.Data)
			}
		}
	}
}

func (p *Player) handlePropertyChange(name string, data interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch name {
	case "time-pos":
		if val, ok := data.(float64); ok {
			p.position = val
		}
	case "duration":
		if val, ok := data.(float64); ok {
			p.duration = val
		}
	case "pause":
		if val, ok := data.(bool); ok {
			p.isPaused = val
		}
	case "eof-reached":
		if val, ok := data.(bool); ok && val {
			if p.OnTrackEnd != nil {
				go p.OnTrackEnd()
			}
		}
	case "audio-codec-name":
		if val, ok := data.(string); ok {
			p.audioCodec = val
		}
	case "audio-bitrate":
		if val, ok := data.(float64); ok {
			p.audioBitrate = int(val / 1000.0)
		}
	case "audio-params/samplerate":
		if val, ok := data.(float64); ok {
			p.audioSampleRate = int(val)
		}
	case "audio-params/channel-count":
		if val, ok := data.(float64); ok {
			p.audioChannels = int(val)
		}
	}

	if p.OnStatus != nil {
		go p.OnStatus()
	}
}

func (p *Player) LoadPlaylist(tracks []string) {
	p.mu.Lock()
	p.playlist = tracks
	p.currentIndex = 0
	p.mu.Unlock()
	p.rebuildShuffle()
}

func (p *Player) Play(index ...int) {
	p.mu.Lock()
	if len(p.playlist) == 0 {
		p.mu.Unlock()
		return
	}

	targetIndex := p.currentIndex
	if len(index) > 0 {
		targetIndex = index[0]
	}
	if targetIndex < 0 || targetIndex >= len(p.playlist) {
		targetIndex = 0
	}

	p.currentIndex = targetIndex
	if p.shuffle && len(p.shuffleOrder) > 0 {
		for i, sIdx := range p.shuffleOrder {
			if sIdx == targetIndex {
				p.shufflePos = i
				break
			}
		}
	}
	trackPath := p.playlist[p.currentIndex]
	p.isPaused = false
	p.stopped = false
	p.mu.Unlock()

	_ = p.sendIPC("loadfile", trackPath, "replace")
	_ = p.sendIPC("set_property", "pause", false)
}

func (p *Player) TogglePause() {
	p.mu.Lock()
	p.isPaused = !p.isPaused
	isPaused := p.isPaused
	p.mu.Unlock()

	_ = p.sendIPC("set_property", "pause", isPaused)
}

func (p *Player) Stop() {
	p.mu.Lock()
	p.stopped = true
	p.position = 0
	p.mu.Unlock()

	_ = p.sendIPC("stop")
}

func (p *Player) Next() {
	p.mu.Lock()
	if len(p.playlist) == 0 {
		p.mu.Unlock()
		return
	}

	if p.repeatMode == "one" {
		p.mu.Unlock()
		p.Play()
		return
	}

	if len(p.manualQueue) > 0 {
		nextIdx := p.manualQueue[0]
		p.manualQueue = p.manualQueue[1:]
		p.mu.Unlock()
		p.Play(nextIdx)
		return
	}

	if p.shuffle {
		p.shufflePos++
		if p.shufflePos >= len(p.shuffleOrder) {
			if p.repeatMode == "all" {
				p.mu.Unlock()
				p.rebuildShuffle()
				p.mu.Lock()
				p.shufflePos = 0
			} else {
				p.mu.Unlock()
				return
			}
		}
		nextIdx := p.shuffleOrder[p.shufflePos]
		p.mu.Unlock()
		p.Play(nextIdx)
		return
	}

	nxt := p.currentIndex + 1
	if nxt >= len(p.playlist) {
		if p.repeatMode == "all" {
			nxt = 0
		} else {
			p.mu.Unlock()
			return
		}
	}
	p.mu.Unlock()
	p.Play(nxt)
}

func (p *Player) Previous() {
	p.mu.Lock()
	if len(p.playlist) == 0 {
		p.mu.Unlock()
		return
	}

	if p.shuffle {
		p.shufflePos--
		if p.shufflePos < 0 {
			p.shufflePos = 0
		}
		prevIdx := p.shuffleOrder[p.shufflePos]
		p.mu.Unlock()
		p.Play(prevIdx)
		return
	}

	prevIdx := (p.currentIndex - 1 + len(p.playlist)) % len(p.playlist)
	p.mu.Unlock()
	p.Play(prevIdx)
}

func (p *Player) Seek(seconds float64, relative bool) {
	if relative {
		_ = p.sendIPC("seek", seconds, "relative")
	} else {
		_ = p.sendIPC("seek", seconds, "absolute")
	}
}

func (p *Player) SetVolume(vol int) {
	if vol < 0 {
		vol = 0
	}
	if vol > 150 {
		vol = 150
	}
	p.mu.Lock()
	p.volume = vol
	p.mu.Unlock()

	_ = p.sendIPC("set_property", "volume", vol)
}

func (p *Player) CycleRepeat() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.repeatMode {
	case "off":
		p.repeatMode = "all"
	case "all":
		p.repeatMode = "one"
	default:
		p.repeatMode = "off"
	}
	return p.repeatMode
}

func (p *Player) ToggleShuffle() bool {
	p.mu.Lock()
	p.shuffle = !p.shuffle
	isShuffle := p.shuffle
	p.mu.Unlock()

	if isShuffle {
		p.rebuildShuffle()
	}
	return isShuffle
}

func (p *Player) rebuildShuffle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.playlist)
	if n == 0 {
		return
	}

	indices := rand.Perm(n)
	p.shuffleOrder = indices
	p.shufflePos = 0
	for i, idx := range indices {
		if idx == p.currentIndex {
			p.shufflePos = i
			break
		}
	}
}

func (p *Player) AddToQueue(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index >= 0 && index < len(p.playlist) {
		p.manualQueue = append(p.manualQueue, index)
	}
}

func (p *Player) GetUpcoming(count int) []int {
	p.mu.Lock()
	defer p.mu.Unlock()

	var result []int
	for _, idx := range p.manualQueue {
		if len(result) >= count {
			break
		}
		result = append(result, idx)
	}

	remaining := count - len(result)
	if remaining <= 0 {
		return result
	}

	if p.shuffle && len(p.shuffleOrder) > 0 {
		for i := 1; len(result) < count && i <= len(p.shuffleOrder); i++ {
			sPos := p.shufflePos + i
			if p.repeatMode == "all" {
				sPos = sPos % len(p.shuffleOrder)
			}
			if sPos >= 0 && sPos < len(p.shuffleOrder) {
				result = append(result, p.shuffleOrder[sPos])
			} else {
				break
			}
		}
	} else {
		for i := 1; i <= remaining; i++ {
			idx := p.currentIndex + i
			if p.repeatMode == "all" && len(p.playlist) > 0 {
				idx = idx % len(p.playlist)
			}
			if idx >= 0 && idx < len(p.playlist) {
				result = append(result, idx)
			}
		}
	}

	return result
}

func (p *Player) GetUpcomingTrackPaths(count int) []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var result []string
	if len(p.playlist) == 0 {
		return result
	}

	// 1. Add manual queue tracks
	for _, idx := range p.manualQueue {
		if len(result) >= count {
			break
		}
		if idx >= 0 && idx < len(p.playlist) {
			result = append(result, p.playlist[idx])
		}
	}

	remaining := count - len(result)
	if remaining <= 0 {
		return result
	}

	// 2. Add upcoming shuffle or sequential tracks
	if p.shuffle && len(p.shuffleOrder) > 0 {
		for i := 1; len(result) < count && i <= len(p.shuffleOrder); i++ {
			sPos := p.shufflePos + i
			if p.repeatMode == "all" {
				sPos = sPos % len(p.shuffleOrder)
			}
			if sPos >= 0 && sPos < len(p.shuffleOrder) {
				idx := p.shuffleOrder[sPos]
				if idx >= 0 && idx < len(p.playlist) {
					result = append(result, p.playlist[idx])
				}
			} else {
				break
			}
		}
	} else {
		for i := 1; i <= remaining && len(result) < count; i++ {
			idx := p.currentIndex + i
			if p.repeatMode == "all" && len(p.playlist) > 0 {
				idx = idx % len(p.playlist)
			}
			if idx >= 0 && idx < len(p.playlist) {
				result = append(result, p.playlist[idx])
			}
		}
	}

	return result
}

// Telemetry getters
func (p *Player) Telemetry() (codec string, bitrate int, sampleRate int, channels int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.audioCodec, p.audioBitrate, p.audioSampleRate, p.audioChannels
}

func (p *Player) TelemetryString() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	parts := []string{}
	if p.audioCodec != "" {
		parts = append(parts, p.audioCodec)
	}
	if p.audioBitrate > 0 {
		parts = append(parts, fmt.Sprintf("%d kbps", p.audioBitrate))
	}
	if p.audioSampleRate > 0 {
		parts = append(parts, fmt.Sprintf("%.1f kHz", float64(p.audioSampleRate)/1000.0))
	}
	if p.audioChannels > 0 {
		if p.audioChannels == 2 {
			parts = append(parts, "stereo")
		} else if p.audioChannels == 1 {
			parts = append(parts, "mono")
		} else {
			parts = append(parts, fmt.Sprintf("%d ch", p.audioChannels))
		}
	}

	if len(parts) == 0 {
		return "—"
	}

	result := ""
	for i, part := range parts {
		if i > 0 {
			result += " · "
		}
		result += part
	}
	return result
}

// Getters for thread-safe state access
func (p *Player) Position() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.position
}

func (p *Player) Duration() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.duration
}

func (p *Player) Volume() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *Player) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isPaused
}

func (p *Player) RepeatMode() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.repeatMode
}

func (p *Player) IsShuffle() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shuffle
}

func (p *Player) CurrentTrackPath() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.currentIndex >= 0 && p.currentIndex < len(p.playlist) {
		return p.playlist[p.currentIndex]
	}
	return ""
}

func (p *Player) CurrentIndex() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentIndex
}

func (p *Player) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil {
		_ = p.conn.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = os.Remove(p.sockPath)
}
