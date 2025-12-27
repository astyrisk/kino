// KINO: Film in terminal
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
	"time"

	"github.com/StalkR/imdb"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"imdb/player"
	"imdb/stream"
)

var cacheSize = flag.String("cache", "30MiB", "Cache size limit for mpv (e.g., 30MiB, 50MiB)")

func main() {
	flag.Parse()
	
	client := &http.Client{
		Transport: &customTransport{http.DefaultTransport},
	}

	if flag.NArg() == 0 {
		interactiveSearch(client)
		return
	}

	// non-interactive mode - pass argument to interactive flow
	query := flag.Arg(0)
	
	// Search for titles
	results, err := imdb.SearchTitle(client, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "No results found.\n")
		os.Exit(3)
	}

	// Select a title using fuzzy finder
	selectedTitle, err := selectTitle(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(4)
	}

	// Check if it's a TV show and handle nested selection
	finalID, err := handleTitleSelection(client, selectedTitle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(5)
	}

	fmt.Printf("\nSelected IMDb ID: %s\n\n", finalID)
	
	// Handle streaming selection
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
		
		// Read input with bufio to handle multi-word queries
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

		// Select a title using fuzzy finder
		selectedTitle, err := selectTitle(results)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("Search cancelled.")
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		// Check if it's a TV show and handle nested selection
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
		
		// Handle streaming selection
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

func selectTitle(titles []imdb.Title) (*imdb.Title, error) {
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

		selectedSeason, err := selectSeason(seasons)
		if err != nil {
			if err.Error() == "abort" {
				// User selected "Go Back" from season selection, return to title selection
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
					// User selected "Go Back" from episode selection, return to season selection
					break
				}
				return "", err
			}

			return fmt.Sprintf("%s/%d-%d", title.ID, selectedSeason, selectedEpisode), nil
		}
	}
}

// getSeasons retrieves available seasons for a TV show
func getSeasons(client *http.Client, title *imdb.Title) ([]int, error) {
	// Use the SeasonCount field from the Title struct
	if title.SeasonCount <= 0 {
		return []int{}, nil
	}
	
	seasons := make([]int, title.SeasonCount)
	for i := 0; i < title.SeasonCount; i++ {
		seasons[i] = i + 1
	}
	
	return seasons, nil
}

// getEpisodes retrieves episodes for a specific season
func getEpisodes(client *http.Client, titleID string, season int) ([]int, error) {
	// Use the imdb.NewSeason function to get season details
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
	// Create a slice with seasons and a "Go Back" option
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

	// Check if "Go Back" was selected
	if idx == len(items)-1 {
		return 0, fmt.Errorf("abort")
	}

	return seasons[idx], nil
}

func selectEpisode(episodes []int) (int, error) {
	// Create a slice with episodes and a "Go Back" option
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

	// Check if "Go Back" was selected
	if idx == len(items)-1 {
		return 0, fmt.Errorf("abort")
	}

	return episodes[idx], nil
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
	
	selectedVariant, err := selectStreamVariant(variants)
	if err != nil {
		return err
	}
	
	fmt.Printf("\nPlaying %s...\n", stream.FormatVariantDisplay(*selectedVariant))
	
	// Get title information for the player window
	title := getTitleForPlayer(imdbID, mediaType, season, episode)
	
	// Create player with cache size and title
	player, err := player.New()
	if err != nil {
		return fmt.Errorf("failed to create player: %w", err)
	}
	
	player.CacheSize = *cacheSize
	player.Title = title
	
	// Play the stream
	err = player.Play(selectedVariant.URL)
	if err != nil {
		return fmt.Errorf("failed to play stream: %w", err)
	}
	
	return nil
}

func parseIMDbID(imdbID string) (stream.MediaType, int, int) {
	// Check if it's a TV episode (format: tt1234567/1-2)
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

func selectStreamVariant(variants []stream.StreamVariant) (*stream.StreamVariant, error) {
	idx, err := fuzzyfinder.Find(
		variants,
		func(i int) string {
			return stream.FormatVariantDisplay(variants[i])
		},
		fuzzyfinder.WithPromptString("Select stream quality:"),
	)
	
	if err != nil {
		return nil, err
	}
	
	return &variants[idx], nil
}

// getTitleForPlayer formats the title for the MPV player window
func getTitleForPlayer(imdbID string, mediaType stream.MediaType, season, episode int) string {
	// Create a client for fetching title info
	client := &http.Client{
		Transport: &customTransport{http.DefaultTransport},
	}
	
	// Extract the base IMDb ID (remove season/episode info)
	baseID := imdbID
	if strings.Contains(imdbID, "/") {
		baseID = strings.Split(imdbID, "/")[0]
	}
	
	// Fetch title information from IMDb
	titleInfo, err := imdb.NewTitle(client, baseID)
	if err != nil {
		log.Printf("Warning: Could not fetch title info: %v", err)
		return "Kino Player"
	}
	
	playerTitle := titleInfo.Name
	
	// Add season and episode information for TV shows
	if mediaType == stream.TV {
		if season > 0 && episode > 0 {
			playerTitle = fmt.Sprintf("%s - S%02dE%02d", titleInfo.Name, season, episode)
		} else if season > 0 {
			playerTitle = fmt.Sprintf("%s - Season %d", titleInfo.Name, season)
		}
	}
	
	// Add year if available
	if titleInfo.Year > 0 {
		playerTitle = fmt.Sprintf("%s (%d)", playerTitle, titleInfo.Year)
	}
	
	return playerTitle
}

// IMDb deployed awswaf and denies requests using the default Go user-agent (Go-http-client/1.1).
// For now it still allows requests from a browser user-agent. Remain respectful, no spam, etc.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	defer time.Sleep(time.Second)         // don't go too fast or risk being blocked by awswaf
	r.Header.Set("Accept-Language", "en") // avoid IP-based language detection
	r.Header.Set("User-Agent", userAgent)
	return e.RoundTripper.RoundTrip(r)
}