package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"kino/extractor"

	fzf "github.com/junegunn/fzf/src"
)

type IMDBResponse struct {
	D []IMDBItem `json:"d"`
}

type IMDBItem struct {
	ID    string `json:"id"`
	Title string `json:"l"`
	QID   string `json:"qid"`
	Year  int    `json:"y"`
}

type TVMazeShow struct {
	ID int `json:"id"`
}

type TVMazeSeason struct {
	ID           int    `json:"id"`
	Number       int    `json:"number"`
	EpisodeOrder int    `json:"episodeOrder"`
	Name         string `json:"name"`
}

type TVMazeEpisode struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Name   string `json:"name"`
}

func main() {
	var query string
	if len(os.Args) >= 2 {
		query = os.Args[1]
	} else {
		fmt.Print("Search: ")
		fmt.Scan(&query)
	}

	if _, err := exec.LookPath("mpv"); err != nil {
		fmt.Println("mpv is not installed")
		os.Exit(1)
	}

	IMDBItems, err := searchIMDB(query)
	if err != nil {
		log.Fatal(err)
	}

	selectedIMDBItem, err := fuzzySelect(IMDBItems, func(item IMDBItem) string {
		return fmt.Sprintf("%s (%d)", item.Title, item.Year)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	season, episode := 0, 0
	mediaType := extractor.Movie
	episodeInfo := ""

	if selectedIMDBItem.QID == "tvSeries" {
		season, episode = handleTV(selectedIMDBItem)
		mediaType = extractor.TV
		episodeInfo = fmt.Sprintf("S%02dE%02d", season, episode)
		if season == 0 && episode == 0 {
			fmt.Fprintln(os.Stderr, "failed to select season/episode")
			os.Exit(1)
		}
	}

	opts := extractor.ResolveOptions{
		IMDBID:  selectedIMDBItem.ID,
		Type:    mediaType,
		Season:  season,
		Episode: episode,
	}

	variants, err := opts.ResolveStreamVariants()
	if err != nil {
		log.Fatal(err)
	}

	selectedVariant, err := fuzzySelect(variants, func(v extractor.StreamVariant) string {
		return extractor.FormatResolutionQuality(v.Resolution)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	title := fmt.Sprintf("%s (%d) %s", selectedIMDBItem.Title, selectedIMDBItem.Year, episodeInfo)
	cmd := exec.Command("mpv",
		"--title="+title,
		"--force-media-title="+title,
		"-demuxer-max-bytes=50MiB",
		selectedVariant.URL,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func searchIMDB(query string) ([]IMDBItem, error) {
	apiURL := fmt.Sprintf("https://v2.sg.media-imdb.com/suggestion/x/%s.json", url.QueryEscape(query))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var imdbResp IMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&imdbResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	filtered := imdbResp.D[:0]
	for _, item := range imdbResp.D {
		switch item.QID {
		case "movie":
			item.Title += " (movie)"
			filtered = append(filtered, item)
		case "tvSeries":
			item.Title += " (TV)"
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func handleTV(imdb IMDBItem) (int, int) {
	var show TVMazeShow
	if err := fetchJSON(fmt.Sprintf("https://api.tvmaze.com/lookup/shows?imdb=%s", imdb.ID), &show); err != nil {
		log.Printf("failed to fetch show: %v", err)
		return 0, 0
	}

	var seasons []TVMazeSeason
	if err := fetchJSON(fmt.Sprintf("https://api.tvmaze.com/shows/%d/seasons", show.ID), &seasons); err != nil {
		log.Printf("failed to fetch seasons: %v", err)
		return 0, 0
	}

	season, err := fuzzySelect(seasons, func(s TVMazeSeason) string {
		return fmt.Sprintf("Season %d (%d episodes)", s.Number, s.EpisodeOrder)
	})
	if err != nil {
		log.Printf("season selection failed: %v", err)
		return 0, 0
	}

	var episodes []TVMazeEpisode
	if err := fetchJSON(fmt.Sprintf("https://api.tvmaze.com/seasons/%d/episodes", season.ID), &episodes); err != nil {
		log.Printf("failed to fetch episodes: %v", err)
		return 0, 0
	}

	episode, err := fuzzySelect(episodes, func(e TVMazeEpisode) string {
		return fmt.Sprintf("E%02d – %s", e.Number, e.Name)
	})
	if err != nil {
		log.Printf("episode selection failed: %v", err)
		return 0, 0
	}

	return season.Number, episode.Number
}

func fetchJSON(url string, target any) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func fuzzySelect[T any](items []T, display func(T) string) (T, error) {
	var zero T

	displays := make([]string, len(items))
	index := make(map[string]T, len(items))
	for i, item := range items {
		d := display(item)
		displays[i] = d
		index[d] = item
	}

	inputChan := make(chan string)
	go func() {
		for _, d := range displays {
			inputChan <- d
		}
		close(inputChan)
	}()

	outputChan := make(chan string, 1)
	opts, err := fzf.ParseOptions(true, []string{})
	if err != nil {
		return zero, err
	}
	opts.Input = inputChan
	opts.Output = outputChan

	code, err := fzf.Run(opts)
	if code == 130 {
		os.Exit(0)
	}
	if err != nil {
		return zero, fmt.Errorf("fzf exited with code %d: %w", code, err)
	}
	return index[<-outputChan], nil
}
