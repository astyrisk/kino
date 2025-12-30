package selection

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/StalkR/imdb"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func HandleTitleSelection(client *http.Client, title *imdb.Title) (string, error) {
	fullTitle, err := imdb.NewTitle(client, title.ID)
	if err != nil {
		return "", fmt.Errorf("error getting title details: %v", err)
	}

	if isTVShow(fullTitle) {
		return handleTVShowSelection(client, fullTitle)
	}

	return title.ID, nil
}

func isTVShow(title *imdb.Title) bool {
	return strings.Contains(strings.ToLower(title.Type), "tv") ||
	       title.SeasonCount > 0
}

func handleTVShowSelection(client *http.Client, title *imdb.Title) (string, error) {
	fmt.Println("\nTV Series detected!")
	
	for {
		seasons, err := getSeasons(client, title)
		if err != nil {
			return "", fmt.Errorf("error getting seasons: %v", err)
		}

		if len(seasons) == 0 {
			fmt.Println("No seasons found. Returning main title ID.")
			return title.ID, nil
		}

		selectedSeason, err := selectSeason(seasons)
		if err != nil {
			if err.Error() == "abort" {
				return "", fmt.Errorf("abort")
			}
			return "", err
		}

		for {
			episodes, err := getEpisodes(client, title.ID, selectedSeason)
			if err != nil {
				return "", fmt.Errorf("error getting episodes: %v", err)
			}

			if len(episodes) == 0 {
				fmt.Println("No episodes found. Returning season ID.")
				return fmt.Sprintf("%s/%d-1", title.ID, selectedSeason), nil
			}

			selectedEpisode, err := selectEpisode(episodes)
			if err != nil {
				if err.Error() == "abort" {
					break
				}
				return "", err
			}

			return fmt.Sprintf("%s/%d-%d", title.ID, selectedSeason, selectedEpisode), nil
		}
	}
}

func getSeasons(client *http.Client, title *imdb.Title) ([]int, error) {
	if title.SeasonCount <= 0 {
		return []int{}, nil
	}
	
	seasons := make([]int, title.SeasonCount)
	for i := 0; i < title.SeasonCount; i++ {
		seasons[i] = i + 1
	}
	
	return seasons, nil
}

func getEpisodes(client *http.Client, titleID string, season int) ([]int, error) {
	seasonInfo, err := imdb.NewSeason(client, titleID, season)
	if err != nil {
		return nil, fmt.Errorf("error getting season %d: %v", season, err)
	}
	
	if len(seasonInfo.Episodes) == 0 {
		return []int{}, nil
	}
	
	episodes := make([]int, len(seasonInfo.Episodes))
	for i, episode := range seasonInfo.Episodes {
		episodes[i] = episode.EpisodeNumber
	}
	
	return episodes, nil
}

func selectSeason(seasons []int) (int, error) {
	items := make([]string, len(seasons)+1)
	for i, season := range seasons {
		items[i] = fmt.Sprintf("Season %d", season)
	}
	items[len(seasons)] = "← Go Back"
	
	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i]
		},
		fuzzyfinder.WithPromptString("Select a season:"),
	)
	
	if err != nil {
		return 0, err
	}

	if idx == len(items)-1 {
		return 0, fmt.Errorf("abort")
	}

	return seasons[idx], nil
}

func selectEpisode(episodes []int) (int, error) {
	items := make([]string, len(episodes)+1)
	for i, episode := range episodes {
		items[i] = fmt.Sprintf("Episode %d", episode)
	}
	items[len(episodes)] = "← Go Back"
	
	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i]
		},
		fuzzyfinder.WithPromptString("Select an episode:"),
	)
	
	if err != nil {
		return 0, err
	}

	if idx == len(items)-1 {
		return 0, fmt.Errorf("abort")
	}

	return episodes[idx], nil
}
