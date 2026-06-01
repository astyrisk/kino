package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"kino/extractor"

	fzf "github.com/junegunn/fzf/src"
)

var histFile string

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
	if err := run(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// run wires up the CLI: it verifies dependencies, parses flags, resolves a
// stream (by searching IMDB or resuming from watch history), lets the user pick
// a quality variant, and plays it in mpv.
//
// TODO: implement the -d (download) flag; it currently returns an error.
func run() error {
	deps := []string{"mpv"}

	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("dependency is not installed: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error reading home directory: %w", err)
	}
	histFile = filepath.Join(homeDir, ".local", "state", "kino-hsts.txt")
	continueFlag := flag.Bool("c", false, "continue watching TV shows")
	downloadFlag := flag.Bool("d", false, "download film/TV episode")
	flag.Parse()

	if *downloadFlag {
		return fmt.Errorf("download is not supported yet")
	}

	var opts extractor.ResolveOptions
	if *continueFlag {
		progresses := LoadAllProgress()
		opts, err = fuzzySelect(progresses, func(progress extractor.ResolveOptions) string {
			return fmt.Sprintf("%s - S%02dE%02d", progress.Title, progress.Season, progress.Episode)
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		opts = userInputOPT()
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
	OpenInMPV(selectedVariant.URL, opts)
	return nil
}

func OpenInMPV(URL string, opts extractor.ResolveOptions) {
	fmt.Println("Requesting URL:", URL)

	title := fmt.Sprintf("%s (%d)", opts.Title, opts.Year)
	if opts.Type == extractor.TV {
		title = fmt.Sprintf("%s (%d) S%02dE%02d", opts.Title, opts.Year, opts.Season, opts.Episode)
	}

	cmd := exec.Command("mpv",
		"--title="+title,
		"--force-media-title="+title,
		"-demuxer-max-bytes=50MiB",
		"--http-header-fields=Referer: https://cloudnestra.com/",
		"--http-header-fields=Origin: https://cloudnestra.com",
		URL,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func userInputOPT() extractor.ResolveOptions {
	var query string
	if flag.NArg() >= 1 {
		query = flag.Arg(0)
	} else {
		fmt.Print("Search: ")
		fmt.Scan(&query)
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

	if selectedIMDBItem.QID == "tvSeries" {
		season, episode = handleTV(selectedIMDBItem)
		mediaType = extractor.TV

		if season == 0 && episode == 0 {
			fmt.Fprintln(os.Stderr, "failed to select season/episode")
			os.Exit(1)
		}

		progress := extractor.ResolveOptions{
			IMDBID:  selectedIMDBItem.ID,
			Title:   selectedIMDBItem.Title,
			Year:    selectedIMDBItem.Year,
			Type:    extractor.TV,
			Season:  season,
			Episode: episode,
		}
		SaveProgress(progress)
	}

	opts := extractor.ResolveOptions{
		IMDBID:  selectedIMDBItem.ID,
		Title:   selectedIMDBItem.Title,
		Year:    selectedIMDBItem.Year,
		Type:    mediaType,
		Season:  season,
		Episode: episode,
	}

	return opts
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

// SaveProgress records the user's current position for a TV show in histFile,
// replacing any existing entry for the same IMDB ID.
//
// TODO: store the *next* episode to resume from rather than the one just watched:
//   - advance to the next episode in the same season when one exists
//   - roll over to the first episode of the next season at a season boundary
//   - mark the show complete (or drop the entry) after the series finale
func SaveProgress(progress extractor.ResolveOptions) {
	line := fmt.Sprintf("%s|%s|%d|%s|%d|%d", progress.IMDBID, progress.Title, progress.Year, progress.Type, progress.Season, progress.Episode)

	data, err := os.ReadFile(histFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	found := false
	for i, l := range lines {
		if strings.HasPrefix(l, progress.IMDBID+"|") {
			lines[i] = line
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, line)
	}

	file, err := os.OpenFile(histFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	file.WriteString(strings.Join(lines, "\n") + "\n")
}

func LoadAllProgress() []extractor.ResolveOptions {
	data, err := os.ReadFile(histFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Fatal(err)
	}

	var progresses []extractor.ResolveOptions
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 6 {
			continue
		}
		year, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		season, err := strconv.Atoi(parts[4])
		if err != nil {
			continue
		}
		episode, err := strconv.Atoi(parts[5])
		if err != nil {
			continue
		}
		progresses = append(progresses, extractor.ResolveOptions{
			IMDBID:  parts[0],
			Title:    parts[1],
			Year:    year,
			Type:    extractor.MediaType(parts[3]),
			Season:  season,
			Episode: episode,
		})
	}

	return progresses
}
