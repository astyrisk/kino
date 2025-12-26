package player

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Player handles video playback
type Player struct {
	playerPath string
}

// New creates a new Player instance
func New() (*Player, error) {
	playerPath, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}
	
	return &Player{
		playerPath: playerPath,
	}, nil
}

// Play starts playing the given URL with mpv
func (p *Player) Play(url string) error {
	log.Printf("Playing stream with mpv: %s", url)
	
	cmd := exec.Command(p.playerPath, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to play stream: %w", err)
	}
	
	return nil
}

// IsAvailable checks if mpv is available on the system
func IsAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

// PlayURL plays a streaming URL with mpv (convenience function)
func PlayURL(url string) error {
	player, err := New()
	if err != nil {
		return err
	}
	
	return player.Play(url)
}

// FormatStreamURL ensures the URL is properly formatted for mpv
func FormatStreamURL(url string) string {
	// Trim any whitespace
	url = strings.TrimSpace(url)
	
	// Ensure it's a valid URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return url
	}
	
	return url
}