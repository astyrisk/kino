package player

import (
	"fmt"
	"os"
	"os/exec"
)

type Player struct {
	playerPath string
	CacheSize  string
	Title      string
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
	args := []string{}

	if p.CacheSize != "" {
		args = append(args, fmt.Sprintf("--demuxer-max-bytes=%s", p.CacheSize))
	}

	if p.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", p.Title))
		args = append(args, fmt.Sprintf("--force-media-title=%s", p.Title))
	}

	args = append(args, url)

	cmd := exec.Command(p.playerPath, args...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
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
