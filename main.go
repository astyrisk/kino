// KINO: Film in terminal
package main

import (
	"flag"
	"fmt"
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

func main() {
	flag.Parse()
	
	client := &http.Client{
		Transport: &customTransport{http.DefaultTransport},
	}

	// If no command-line arguments, run in interactive mode
	if flag.NArg() == 0 {
		interactiveSearch(client)
		return
	}

	// Original non-interactive mode for backward compatibility
	title, err := imdb.SearchTitle(client, flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	if len(title) == 0 {
		fmt.Fprintf(os.Stderr, "Not found.")
		os.Exit(3)
	}
	
	// Print top 5 results
	fmt.Println("Top 5 Results:")
	fmt.Println("-------------")
	for i := 0; i < len(title) && i < 5; i++ {
		year := ""
		if title[i].Year > 0 {
			year = fmt.Sprintf(" (%d)", title[i].Year)
		}
		fmt.Printf("%d. %s%s - ID: %s\n", i+1, title[i].Name, year, title[i].ID)
	}
	
	// Get and display details for the first result
	t, err := imdb.NewTitle(client, title[0].ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}

	fmt.Println("\nDetails for first result:")
	fmt.Println("------------------------")
	fmt.Println(t.String())
}

// interactiveSearch runs the interactive fuzzy finder mode
func interactiveSearch(client *http.Client) {
	// Set up signal handler for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	fmt.Println("Interactive IMDb Search")
	fmt.Println("======================")
	fmt.Println("Enter a film or TV show title (Ctrl+C to exit)")
	fmt.Println()

	for {
		// Get search query from user
		fmt.Print("Search: ")
		var query string
		_, err := fmt.Scanln(&query)
		if err != nil {
			// Handle Ctrl+C or EOF
			if err.Error() == "interrupt" || err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		// Search for titles
		results, err := imdb.SearchTitle(client, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error searching: %v\n", err)
			continue
		}

		if len(results) == 0 {
			fmt.Println("No results found. Try a different search.")
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

// selectTitle shows a fuzzy finder for title selection
func selectTitle(titles []imdb.Title) (*imdb.Title, error) {
	idx, err := fuzzyfinder.Find(
		titles,
		func(i int) string {
			title := titles[i]
			year := ""
			if title.Year > 0 {
				year = fmt.Sprintf(" (%d)", title.Year)
			}
			// Determine if it's TV or Film
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

// handleTitleSelection handles the selection process for a title
func handleTitleSelection(client *http.Client, title *imdb.Title) (string, error) {
	// Get detailed title information to check if it's a TV show
	fullTitle, err := imdb.NewTitle(client, title.ID)
	if err != nil {
		return "", fmt.Errorf("error getting title details: %v", err)
	}

	// Check if it's a TV show (has seasons and episodes)
	if isTVShow(fullTitle) {
		return handleTVShowSelection(client, fullTitle)
	}

	// For movies, return the title ID directly
	return title.ID, nil
}

// isTVShow checks if a title is a TV series with seasons
func isTVShow(title *imdb.Title) bool {
	// Check if it's a TV series based on Type field or SeasonCount
	return strings.Contains(strings.ToLower(title.Type), "tv") ||
	       title.SeasonCount > 0
}

// handleTVShowSelection handles nested selection for TV shows
func handleTVShowSelection(client *http.Client, title *imdb.Title) (string, error) {
	fmt.Println("\nTV Series detected!")
	
	// Get seasons
	seasons, err := getSeasons(client, title.ID)
	if err != nil {
		return "", fmt.Errorf("error getting seasons: %v", err)
	}

	if len(seasons) == 0 {
		fmt.Println("No seasons found. Returning main title ID.")
		return title.ID, nil
	}

	// Select season
	selectedSeason, err := selectSeason(seasons)
	if err != nil {
		return "", err
	}

	// Get episodes for selected season
	episodes, err := getEpisodes(client, title.ID, selectedSeason)
	if err != nil {
		return "", fmt.Errorf("error getting episodes: %v", err)
	}

	if len(episodes) == 0 {
		fmt.Println("No episodes found. Returning season ID.")
		return fmt.Sprintf("%s/season/%d", title.ID, selectedSeason), nil
	}

	// Select episode
	selectedEpisode, err := selectEpisode(episodes)
	if err != nil {
		return "", err
	}

	// Return the full IMDb ID with season and episode
	return fmt.Sprintf("%s/episode/%d/%d", title.ID, selectedSeason, selectedEpisode), nil
}

// getSeasons retrieves available seasons for a TV show
func getSeasons(client *http.Client, titleID string) ([]int, error) {
	// For now, we'll use a simple approach: try to get season information
	// In a real implementation, you would parse the title details
	// Since the imdb package may not directly expose this, we'll use a heuristic
	
	// Try to get additional title information
	// This is a placeholder - you may need to adjust based on the actual imdb package capabilities
	return []int{1, 2, 3, 4, 5}, nil
}

// getEpisodes retrieves episodes for a specific season
func getEpisodes(client *http.Client, titleID string, season int) ([]int, error) {
	// Similar to getSeasons, this would need to be implemented based on the imdb package
	// For demonstration, we'll return a range of episodes
	episodes := make([]int, 10)
	for i := 0; i < 10; i++ {
		episodes[i] = i + 1
	}
	return episodes, nil
}

// selectSeason shows a fuzzy finder for season selection
func selectSeason(seasons []int) (int, error) {
	idx, err := fuzzyfinder.Find(
		seasons,
		func(i int) string {
			return fmt.Sprintf("Season %d", seasons[i])
		},
		fuzzyfinder.WithPromptString("Select a season:"),
	)
	
	if err != nil {
		return 0, err
	}

	return seasons[idx], nil
}

// selectEpisode shows a fuzzy finder for episode selection
func selectEpisode(episodes []int) (int, error) {
	idx, err := fuzzyfinder.Find(
		episodes,
		func(i int) string {
			return fmt.Sprintf("Episode %d", episodes[i])
		},
		fuzzyfinder.WithPromptString("Select an episode:"),
	)
	
	if err != nil {
		return 0, err
	}

	return episodes[idx], nil
}

// handleStreamingSelection handles the streaming workflow
func handleStreamingSelection(imdbID string) error {
	// Check if mpv is available
	if !player.IsAvailable() {
		fmt.Println("\nWarning: mpv not found in PATH")
		fmt.Println("Please install mpv to enable streaming playback")
		fmt.Println("On Ubuntu/Debian: sudo apt install mpv")
		fmt.Println("On macOS: brew install mpv")
		fmt.Println("On Windows: choco install mpv")
		return nil
	}
	
	// Parse IMDb ID to determine media type and extract season/episode
	mediaType, season, episode := parseIMDbID(imdbID)
	
	// Fetch streaming variants
	fmt.Println("\nFetching streaming options...")
	variants, err := stream.GetStreamVariants(imdbID, mediaType, season, episode)
	if err != nil {
		return fmt.Errorf("failed to get streaming variants: %w", err)
	}
	
	if len(variants) == 0 {
		return fmt.Errorf("no streaming variants found")
	}
	
	// Select a variant using fuzzy finder
	selectedVariant, err := selectStreamVariant(variants)
	if err != nil {
		return err
	}
	
	// Play the selected variant
	fmt.Printf("\nPlaying %s...\n", stream.FormatVariantDisplay(*selectedVariant))
	err = player.PlayURL(selectedVariant.URL)
	if err != nil {
		return fmt.Errorf("failed to play stream: %w", err)
	}
	
	return nil
}

// parseIMDbID parses an IMDb ID to extract media type, season, and episode
func parseIMDbID(imdbID string) (stream.MediaType, int, int) {
	// Check if it's a TV episode (format: tt1234567/episode/1/2)
	if strings.Contains(imdbID, "/episode/") {
		parts := strings.Split(imdbID, "/episode/")
		if len(parts) == 2 {
			seParts := strings.Split(parts[1], "/")
			if len(seParts) == 2 {
				season, _ := strconv.Atoi(seParts[0])
				episode, _ := strconv.Atoi(seParts[1])
				return stream.TV, season, episode
			}
		}
	}
	
	// Check if it's a TV season (format: tt1234567/season/1)
	if strings.Contains(imdbID, "/season/") {
		parts := strings.Split(imdbID, "/season/")
		if len(parts) == 2 {
			season, _ := strconv.Atoi(parts[1])
			return stream.TV, season, 1
		}
	}
	
	// Default to movie
	return stream.Movie, 0, 0
}

// selectStreamVariant shows a fuzzy finder for stream variant selection
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