package tui

import (
	"fmt"
	"net/http"

	"strings"

	"github.com/StalkR/imdb"
	"github.com/ktr0731/go-fuzzyfinder"
)

// fuzzyFind is a helper function to reduce duplication in fuzzy finder calls
func fuzzyFind(items []string, prompt string, log *Logger) (int, error) {
	log.Suspend()
	defer log.Resume()

	return fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i]
		},
		fuzzyfinder.WithPromptString(prompt),
	)
}

// ShowPostPlayMenu displays a simple menu like ani-cli
// Returns: "next", "replay", "previous", "select", "change_quality", or "quit"
func ShowPostPlayMenu(mediaType string, hasNext, hasPrev bool, title string, season, episode int, log *Logger) (string, error) {
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

	idx, err := fuzzyFind(options, buildPromptString(mediaType, title, season, episode), log)

	if err != nil {
		return "", err
	}

	return options[idx], nil
}

func buildPromptString(mediaType string, title string, season, episode int) string {
	if mediaType == "tv" && season > 0 && episode > 0 {
		return fmt.Sprintf("Playing episode %d of %s...", episode, title)
	}
	return fmt.Sprintf("Playing %s...", title)
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

func Interactive(client *http.Client, log *Logger) (*imdb.Title, error) {
	for {
		// Show search prompt at bottom
		query := log.PromptAtBottom("Search kino: ")

		if query == "" {
			continue
		}

		// Show searching status
		log.ShowStatus(fmt.Sprintf("Searching for: %s...", query))

		results, err := imdb.SearchTitle(client, query)
		if err != nil {
			log.ShowError(fmt.Sprintf("Error searching: %v", err))
			continue
		}

		if len(results) == 0 {
			log.ShowInfo("No results found.")
			continue
		}

		selectedTitle, err := SelectTitle(results, log)

		if err != nil {
			if err.Error() == "abort" {
				log.ShowInfo("Search cancelled.")
				continue
			}
			log.ShowError(err.Error())
			continue
		}

		return selectedTitle, nil
	}
}

func SelectTitle(titles []imdb.Title, log *Logger) (*imdb.Title, error) {
	log.Suspend()
	defer log.Resume()

	idx, err := fuzzyfinder.Find(
		titles,
		func(i int) string {
			title := titles[i]
			year := ""
			if title.Year > 0 {
				year = fmt.Sprintf(" (%d)", title.Year)
			}
			mediaType := "Film"
			if strings.Contains(strings.ToLower(title.Type), "tv") {
				mediaType = "TV"
			}
			return fmt.Sprintf("%s%s [%s]", title.Name, year, mediaType)
		},
		fuzzyfinder.WithPromptString("Select a title:"),
	)

	if err != nil {
		return nil, err
	}

	return &titles[idx], nil
}

func HandleTitleSelection(client *http.Client, title *imdb.Title, log *Logger) (string, error) {
	fullTitle, err := imdb.NewTitle(client, title.ID)
	if err != nil {
		return "", fmt.Errorf("error getting title details: %v", err)
	}

	if isTVShow(fullTitle) {
		return handleTVShowSelection(client, fullTitle, log)
	}

	return title.ID, nil
}

func isTVShow(title *imdb.Title) bool {
	return strings.Contains(strings.ToLower(title.Type), "tv") ||
		title.SeasonCount > 0
}

func handleTVShowSelection(client *http.Client, title *imdb.Title, log *Logger) (string, error) {
	for {
		seasons, err := getSeasons(client, title)
		if err != nil {
			return "", fmt.Errorf("error getting seasons: %v", err)
		}

		if len(seasons) == 0 {
			log.ShowInfo("No seasons found. Returning main title ID.")
			return title.ID, nil
		}

		selectedSeason, err := selectSeason(seasons, log)
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
				log.ShowInfo("No episodes found. Returning season ID.")
				return fmt.Sprintf("%s/%d-1", title.ID, selectedSeason), nil
			}

			selectedEpisode, err := selectEpisode(episodes, log)
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

func selectSeason(seasons []int, log *Logger) (int, error) {
	items := make([]string, len(seasons)+1)
	for i, season := range seasons {
		items[i] = fmt.Sprintf("Season %d", season)
	}
	items[len(seasons)] = "← Go Back"

	idx, err := fuzzyFind(items, "Select a season:", log)

	if err != nil {
		return 0, err
	}

	if idx == len(items)-1 {
		return 0, fmt.Errorf("abort")
	}

	return seasons[idx], nil
}

func selectEpisode(episodes []int, log *Logger) (int, error) {
	items := make([]string, len(episodes)+1)
	for i, episode := range episodes {
		items[i] = fmt.Sprintf("Episode %d", episode)
	}
	items[len(episodes)] = "← Go Back"

	idx, err := fuzzyFind(items, "Select an episode:", log)

	if err != nil {
		return 0, err
	}

	if idx == len(items)-1 {
		return 0, fmt.Errorf("abort")
	}

	return episodes[idx], nil
}

// func selectEpisode(episodes []int) (int, error) {
// 	items := make([]string, len(episodes)+1)
// 	for i, episode := range episodes {
// 		items[i] = fmt.Sprintf("Episode %d", episode)
// 	}
// 	items[len(episodes)] = "← Go Back"

// 	idx, err := fuzzyFind(items, "Select an episode:")

// 	if err != nil {
// 		return 0, err
// 	}

// 	if idx == len(items)-1 {
// 		return 0, fmt.Errorf("abort")
// 	}

// 	return episodes[idx], nil
// }
