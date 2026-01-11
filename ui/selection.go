package ui

import (
	"fmt"
	"strings"

	"kino/extractor"

	"github.com/StalkR/imdb"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

// SelectTitle prompts the user to select a title from a list of IMDb search results.
func SelectTitle(titles []imdb.Title) (*imdb.Title, error) {
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

// SelectSeason prompts the user to select a season number.
func SelectSeason(seasons []int) (int, error) {
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

// SelectEpisode prompts the user to select an episode number.
func SelectEpisode(episodes []int) (int, error) {
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

// SelectStreamVariant prompts the user to select a stream quality variant.
func SelectStreamVariant(variants []extractor.StreamVariant) (*extractor.StreamVariant, error) {
	idx, err := fuzzyfinder.Find(
		variants,
		func(i int) string {
			return FormatVariantDisplay(variants[i])
		},
		fuzzyfinder.WithPromptString("Select stream quality:"),
	)

	if err != nil {
		return nil, err
	}

	return &variants[idx], nil
}
