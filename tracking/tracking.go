package tracking

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type WatchEntry struct {
	IMDbID    string    `json:"imdb_id"`
	Title     string    `json:"title"`
	MediaType string    `json:"media_type"`
	Season    int       `json:"season,omitempty"`
	Episode   int       `json:"episode,omitempty"`
	WatchedAt time.Time `json:"watched_at"`
	Duration  int       `json:"duration_seconds"`
}

func RecordWatch(imdbID, title, mediaType string, season, episode, duration int) error {
	configDir, err := ensureConfigDir()
	if err != nil {
		return err
	}
	
	entry := WatchEntry{
		IMDbID:    imdbID,
		Title:     title,
		MediaType: mediaType,
		Season:    season,
		Episode:   episode,
		WatchedAt: time.Now(),
		Duration:  duration,
	}
	
	return appendToHistory(configDir, entry)
}

func ensureConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".config", "kino")
	return configDir, os.MkdirAll(configDir, 0755)
}

func appendToHistory(configDir string, entry WatchEntry) error {
	historyFile := filepath.Join(configDir, "watched.json")
	
	var entries []WatchEntry
	if data, err := os.ReadFile(historyFile); err == nil {
		json.Unmarshal(data, &entries)
	}
	
	entries = append(entries, entry)
	
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(historyFile, data, 0644)
}