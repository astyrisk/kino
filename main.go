package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"kino/internal/client"
	"kino/player"
	"kino/stream"
	"kino/ui"

	"github.com/StalkR/imdb"
)

var cacheSize = flag.String("cache", "12MiB", "Cache size limit for mpv (e.g., 30MiB, 50MiB)")

func main() {
	flag.Parse()

	client := client.New()

	if flag.NArg() == 0 {
		interactiveSearch(client)
		return
	}

	query := flag.Arg(0)

	results, err := imdb.SearchTitle(client, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "No results found.\n")
		os.Exit(3)
	}

	selectedTitle, err := ui.SelectTitle(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(4)
	}

	finalID, err := handleTitleSelection(client, selectedTitle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(5)
	}

	fmt.Printf("\nSelected IMDb ID: %s\n\n", finalID)

	err = handleStreamingSelection(finalID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(6)
	}
}

func interactiveSearch(client *http.Client) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Search kino: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		query := strings.TrimSpace(input)
		if query == "" {
			continue
		}

		results, err := imdb.SearchTitle(client, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
			continue
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			fmt.Println()
			continue
		}

		selectedTitle, err := ui.SelectTitle(results)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("Search cancelled.")
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		finalID, err := handleTitleSelection(client, selectedTitle)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("Selection cancelled.")
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Printf("\nSelected IMDb ID: %s\n\n", finalID)

		err = handleStreamingSelection(finalID)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("Streaming cancelled.")
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println("Playback finished. Starting new search...")
		fmt.Println()
	}
}

func handleTitleSelection(client *http.Client, title *imdb.Title) (string, error) {
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

		selectedSeason, err := ui.SelectSeason(seasons)
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

			selectedEpisode, err := ui.SelectEpisode(episodes)
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

func handleStreamingSelection(imdbID string) error {
	if !player.IsAvailable() {
		fmt.Println("\nWarning: mpv not found in PATH")
		fmt.Println("Please install mpv to enable streaming playback")
		fmt.Println("On Ubuntu/Debian: sudo apt install mpv")
		fmt.Println("On macOS: brew install mpv")
		fmt.Println("On Windows: choco install mpv")
		return nil
	}

	mediaType, season, episode := parseIMDbID(imdbID)

	fmt.Println("\nFetching streaming options...")
	variants, err := stream.GetStreamVariants(imdbID, mediaType, season, episode)
	if err != nil {
		return fmt.Errorf("failed to get streaming variants: %w", err)
	}

	if len(variants) == 0 {
		return fmt.Errorf("no streaming variants found")
	}

	selectedVariant, err := ui.SelectStreamVariant(variants)
	if err != nil {
		return err
	}

	fmt.Printf("\nPlaying %s...\n", ui.FormatVariantDisplay(*selectedVariant))

	title := getTitleForPlayer(imdbID, mediaType, season, episode)

	player, err := player.New()
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}

	player.CacheSize = *cacheSize
	player.Title = title

	err = player.Play(selectedVariant.URL)
	if err != nil {
		return fmt.Errorf("failed to play stream: %w", err)
	}

	return nil
}

func parseIMDbID(imdbID string) (stream.MediaType, int, int) {
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

func getTitleForPlayer(imdbID string, mediaType stream.MediaType, season, episode int) string {
	client := client.New()

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
