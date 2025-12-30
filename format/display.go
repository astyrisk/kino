package format

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/StalkR/imdb"
	"imdb/stream"
)

// ParseIMDbID parses an IMDb ID string and returns the media type, season, and episode
func ParseIMDbID(imdbID string) (stream.MediaType, int, int) {
	parts := strings.Split(imdbID, "/")
	if len(parts) == 2 {
		seParts := strings.Split(parts[1], "-")
		if len(seParts) == 2 {
			season, _ := strconv.Atoi(seParts[0])
			episode, _ := strconv.Atoi(seParts[1])
			return stream.TV, season, episode
		}
	}
	
	return stream.Movie, 0, 0
}

// GetTitleInfo retrieves the IMDb title information for the given ID
func GetTitleInfo(client *http.Client, imdbID string) *imdb.Title {
	baseID := imdbID
	if strings.Contains(imdbID, "/") {
		baseID = strings.Split(imdbID, "/")[0]
	}
	
	titleInfo, err := imdb.NewTitle(client, baseID)
	if err != nil {
		log.Printf("Warning: Could not fetch title info: %v", err)
		return &imdb.Title{Name: "Unknown Title"}
	}
	
	return titleInfo
}

// GetTitleForPlayer returns a formatted title string for the mpv player
func GetTitleForPlayer(client *http.Client, imdbID string, mediaType stream.MediaType, season, episode int) string {	
	baseID := imdbID
	if strings.Contains(imdbID, "/") {
		baseID = strings.Split(imdbID, "/")[0]
	}
	
	titleInfo, err := imdb.NewTitle(client, baseID)
	if err != nil {
		log.Printf("Warning: Could not fetch title info: %v", err)
		return "Kino Player"
	}
	
	playerTitle := titleInfo.Name
	
	if mediaType == stream.TV {
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

// GetMediaTypeString converts stream.MediaType to a string representation
func GetMediaTypeString(mediaType stream.MediaType) string {
	if mediaType == stream.TV {
		return "tv"
	}
	return "movie"
}
