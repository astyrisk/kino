package tracking

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RecordWatch records a watch entry in the history file
// Format: episode_number\timdb_id\ttitle (episode_count episodes)
func RecordWatch(imdbID, title, mediaType string, season, episode, duration int) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// Create simple entry format like ani-cli
	// Extract base ID for TV shows (remove season/episode part)
	baseID := imdbID
	if strings.Contains(imdbID, "/") {
		baseID = strings.Split(imdbID, "/")[0]
	}

	// Format episode number for TV shows, use "1" for movies
	epNum := "1"
	if mediaType == "tv" && season > 0 && episode > 0 {
		epNum = fmt.Sprintf("%d", episode)
	}

	return appendToHistory(configDir, baseID, title, epNum)
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(home, ".config", "kino")
	return configDir, os.MkdirAll(configDir, 0755)
}

func appendToHistory(configDir, imdbID, title, episodeNum string) error {
	historyFile := filepath.Join(configDir, "kino-hsts")
	
	// Read existing entries
	var entries []string
	if data, err := os.ReadFile(historyFile); err == nil {
		entries = strings.Split(strings.TrimSpace(string(data)), "\n")
	}
	
	// Remove old entry for this title if exists
	newEntries := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		if entry != "" && !strings.Contains(entry, "\t"+imdbID+"\t") {
			newEntries = append(newEntries, entry)
		}
	}
	
	// Add new entry at the end
	newEntry := fmt.Sprintf("%s\t%s\t%s", episodeNum, imdbID, title)
	newEntries = append(newEntries, newEntry)
	
	// Write back to file
	data := strings.Join(newEntries, "\n")
	if !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	
	return os.WriteFile(historyFile, []byte(data), 0644)
}