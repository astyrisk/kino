package player

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type Player struct {
	playerPath string
	CacheSize  string
	Title      string
	cmd        *exec.Cmd
	mu         sync.Mutex
}

func New() (*Player, error) {
	playerPath, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}

	return &Player{
		playerPath: playerPath,
	}, nil
}

func (p *Player) Play(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	args := []string{}

	if p.CacheSize != "" {
		args = append(args, fmt.Sprintf("--demuxer-max-bytes=%s", p.CacheSize))
	}

	if p.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", p.Title))
		args = append(args, fmt.Sprintf("--force-media-title=%s", p.Title))
	}

	args = append(args, url)

	p.cmd = exec.Command(p.playerPath, args...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	p.cmd.Stdin = os.Stdin

	return p.cmd.Run()
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}

func IsAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

func PlayURL(url string) error {
	player, err := New()
	if err != nil {
		return err
	}

	return player.Play(url)
}
