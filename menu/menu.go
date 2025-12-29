package menu

import (
	"fmt"
	"net/http"
	
	"github.com/StalkR/imdb"
	"github.com/ktr0731/go-fuzzyfinder"
)

type MediaType int

const (
	Movie MediaType = iota
	TVShow
)

func ShowPostPlayMenu(mediaType MediaType, hasNext, hasPrev bool) (string, error) {
	var options []string
	
	if mediaType == TVShow {
		if hasNext {
			options = append(options, "▶ Next Episode")
		}
		options = append(options, "🔄 Replay")
		if hasPrev {
			options = append(options, "◀ Previous Episode")
		}
	} else {
		options = append(options, "🔄 Replay")
	}
	
	options = append(options, "⚙️ Change Quality", "🔍 New Search", "❌ Quit")
	
	idx, err := fuzzyfinder.Find(
		options,
		func(i int) string {
			return options[i]
		},
		fuzzyfinder.WithPromptString("What would you like to do?"),
	)
	
	if err != nil {
		return "", err
	}
	
	return options[idx], nil
}

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