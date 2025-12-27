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

	if flag.NArg() == 0 {
		interactiveSearch(client)
		return
	}

	// non-interactive mode 
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
}

func interactiveSearch(client *http.Client) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		fmt.Print("Search kino: ")
		
		// Use a goroutine to read input non-blockingly
		inputChan := make(chan string)
		errChan := make(chan error)
		
		go func() {
			var query string
			_, err := fmt.Scanln(&query)
			if err != nil {
				errChan <- err
				return
			}
			inputChan <- query
		}()

		// Wait for either input, signal, or error
		select {
		case sig := <-sigChan:
			fmt.Println("Goodbye!")
			return
		case query := <-inputChan:
			query = strings.TrimSpace(query)
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
		case err := <-errChan:
			if err.Error() == "interrupt" || err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}
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
			if isTVShow(title) {
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
	
	seasons, err := getSeasons(client, title.ID)
	if err != nil {
		return "", fmt.Errorf("error getting seasons: %v", err)
	}

	if len(seasons) == 0 {
		fmt.Println("No seasons found. Returning main title ID.")
		return title.ID, nil
	}

	selectedSeason, err := selectSeason(seasons)
	if err != nil {
		return "", err
	}

	episodes, err := getEpisodes(client, title.ID, selectedSeason)
	if err != nil {
		return "", fmt.Errorf("error getting episodes: %v", err)
	}

	if len(episodes) == 0 {
		fmt.Println("No episodes found. Returning season ID.")
		return fmt.Sprintf("%s/season/%d", title.ID, selectedSeason), nil
	}

	selectedEpisode, err := selectEpisode(episodes)
	if err != nil {
		return "", err
	}

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
	err = player.PlayURL(selectedVariant.URL)
	if err != nil {
		return fmt.Errorf("failed to play stream: %w", err)
	}
	
	return nil
}

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