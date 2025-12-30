package tracking

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RecordWatch(imdbID, title, mediaType string, season, episode, duration int) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	baseID := imdbID
	if strings.Contains(imdbID, "/") {
		baseID = strings.Split(imdbID, "/")[0]
	}

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
	stateDir := filepath.Join(home, ".local", "state", "kino")

	return stateDir, os.MkdirAll(stateDir, 0755)
}

func appendToHistory(configDir, imdbID, title, episodeNum string) error {
	historyFile := filepath.Join(configDir, "kino-hsts")
	
	var entries []string
	if data, err := os.ReadFile(historyFile); err == nil {
		entries = strings.Split(strings.TrimSpace(string(data)), "\n")
	}
	
	newEntries := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		if entry != "" && !strings.Contains(entry, "\t"+imdbID+"\t") {
			newEntries = append(newEntries, entry)
		}
	}
	
	newEntry := fmt.Sprintf("%s\t%s\t%s", episodeNum, imdbID, title)
	newEntries = append(newEntries, newEntry)
	
	data := strings.Join(newEntries, "\n")
	if !strings.HasSuffix(data, "\n") {
		data += "\n"
	}
	
	return os.WriteFile(historyFile, []byte(data), 0644)
}