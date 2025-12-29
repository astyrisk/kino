package menu

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	
	"github.com/StalkR/imdb"
	"github.com/ktr0731/go-fuzzyfinder"
)

// ShowPostPlayMenu displays a simple menu like ani-cli
// Returns: "next", "replay", "previous", "select", "change_quality", or "quit"
func ShowPostPlayMenu(mediaType string, hasNext, hasPrev bool) (string, error) {
	var options []string
	
	if mediaType == "tv" {
		if hasNext {
			options = append(options, "next")
		}
		options = append(options, "replay")
		if hasPrev {
			options = append(options, "previous")
		}
	} else {
		options = append(options, "replay")
	}
	
	options = append(options, "select", "change_quality", "quit")
	
	idx, err := fuzzyfinder.Find(
		options,
		func(i int) string {
			return options[i]
		},
		fuzzyfinder.WithPromptString("What would you like to do?"),
		// Playing episode * of * [tv] or playing * [film]
	)
	
	if err != nil {
		return "", err
	}
	
	return options[idx], nil
}

// GetNextEpisode returns the next episode (season, episode)
func GetNextEpisode(client *http.Client, titleID string, season, episode int) (int, int, error) {
	sInfo, err := imdb.NewSeason(client, titleID, season)
	if err != nil {
		return 0, 0, err
	}
	
	if episode < len(sInfo.Episodes) {
		return season, episode + 1, nil
	}
	
	title, err := imdb.NewTitle(client, titleID)
	if err != nil {
		return 0, 0, err
	}
	
	if season >= title.SeasonCount {
		return 0, 0, fmt.Errorf("no next episode")
	}
	
	return season + 1, 1, nil
}

// GetPreviousEpisode returns the previous episode (season, episode)
func GetPreviousEpisode(client *http.Client, titleID string, season, episode int) (int, int, error) {
	if episode > 1 {
		return season, episode - 1, nil
	}
	
	if season == 1 {
		return 0, 0, fmt.Errorf("no previous episode")
	}
	
	prevInfo, err := imdb.NewSeason(client, titleID, season-1)
	if err != nil {
		return 0, 0, err
	}
	
	if len(prevInfo.Episodes) == 0 {
		return 0, 0, fmt.Errorf("no previous episode")
	}
	
	return season - 1, len(prevInfo.Episodes), nil
}

// ParseIMDbID extracts season and episode from IMDb ID
func ParseIMDbID(imdbID string) (int, int) {
	parts := strings.Split(imdbID, "/")
	if len(parts) == 2 {
		seParts := strings.Split(parts[1], "-")
		if len(seParts) == 2 {
			season, _ := strconv.Atoi(seParts[0])
			episode, _ := strconv.Atoi(seParts[1])
			return season, episode
		}
	}
	return 0, 0
}