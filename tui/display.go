package tui

import (
	"fmt"
	"kino/extractor"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/StalkR/imdb"
)

// extractBaseID extracts the base IMDb ID from a potentially compound ID
// Compound IDs have format: "tt1234567/1-2" for TV episodes
func extractBaseID(imdbID string) string {
		return strings.Split(imdbID, "/")[0]
	}
	return imdbID
}

// ParseIMDbID parses a compound IMDb ID to extract media type and episode info
func ParseIMDbID(imdbID string) (extractor.MediaType, int, int) {
	parts := strings.Split(imdbID, "/")
		if len(seParts) == 2 {
			season, _ := strconv.Atoi(seParts[0])
			episode, _ := strconv.Atoi(seParts[1])
			return extractor.TV, season, episode
		}
	}
	return extractor.Movie, 0, 0
}

// GetTitleInfo fetches title information from IMDb
func GetTitleInfo(client *http.Client, imdbID string) *imdb.Title {
	baseID := extractBaseID(imdbID)
	titleInfo, err := imdb.NewTitle(client, baseID)
	if err != nil {
		log.Printf("Warning: Could not fetch title info: %v", err)
		return &imdb.Title{Name: "Unknown Title"}
	}
	return titleInfo
}

// GetTitleForPlayer generates a formatted title for the player
func GetTitleForPlayer(client *http.Client, imdbID string, mediaType extractor.MediaType, season, episode int) string {
	baseID := extractBaseID(imdbID)
	titleInfo, err := imdb.NewTitle(client, baseID)
	if err != nil {
		log.Printf("Warning: Could not fetch title info: %v", err)
		return "Kino Player"
	}

	playerTitle := titleInfo.Name

	if mediaType == extractor.TV {
		if season > 0 && episode > 0 {
			playerTitle = fmt.Sprintf("%s - S%02dE%02d", titleInfo.Name, season, episode)
		} else if season > 0 {
			playerTitle = fmt.Sprintf("%s - Season %d", titleInfo.Name, season)
		}
	}

	if titleInfo.Year > 0 {
		playerTitle = fmt.Sprintf("%s (%d)", playerTitle, titleInfo.Year)
	}

	return playerTitle
}

func GetMediaTypeString(mediaType extractor.MediaType) string {
	if mediaType == extractor.TV {
		return "tv"
	}
	return "movie"
}

func RecordWatch(imdbID, title, mediaType string, season, episode, duration int) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	baseID := extractBaseID(imdbID)
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
