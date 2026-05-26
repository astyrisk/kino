/*kino: streams films right into mpv*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"io"
	"net/http"
	"net/url"
	"encoding/json"
	"log"
	fzf "github.com/junegunn/fzf/src"

	"kino/extractor"
)

type IMDBResponse struct {
	D []IMDBItem `json:"d"`
}

type IMDBItem struct {
	ID 	string	`json:"id"`
	Title 	string	`json:"l"`
	QID 	string	`json:"qid"`
	Year 	int   	`json:"y"`
}

func SearchIMDB(query string) ([]IMDBItem, error) {
	encoded := url.QueryEscape(query)
	// using the IMDB suggestion API "for now"
	apiURL := fmt.Sprintf("https://v2.sg.media-imdb.com/suggestion/x/%s.json", encoded)

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var imdbResp IMDBResponse
	if err := json.Unmarshal(body, &imdbResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	i := 0
	for _, item := range imdbResp.D {
		if item.QID == "movie" {
			imdbResp.D[i] = item
			i++
		}
	}
	return imdbResp.D[:i], nil
}

func FuzzySelect[T any](items []T, display func(T) string) (T, error) {
	var zero T

	displays := make([] string, len(items))
	index := make(map[string]T, len(items))
	for i, item := range items{
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
		return zero, fmt.Errorf("fzf exited with code %d:  %w", code, err)
	}
	return index[<-outputChan], nil
}

func OpenInMPV(url string, item IMDBItem) {
	title := fmt.Sprintf("%s (%d)", item.Title, item.Year)
	titleFlag := "--title=" + title
	forceTitle := "--force-media-title=" + title
	cache := fmt.Sprintf("-demuxer-max-bytes=50MiB")
	cmd := exec.Command("mpv", titleFlag, forceTitle, cache, url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

/*
 * -c: continue a TV
*/
func main() {
	var query string

	if len(os.Args) >= 2 {
		query = os.Args[1];
	} else {
		fmt.Print("Film: ")
		fmt.Scan(&query)
	}

	deps := []string{"mpv"}
	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			fmt.Printf("%s is not installed\n", dep)
			os.Exit(1)
		}
	}
	fmt.Println("All dependencies are installed")

	// fetch the results from imdb
	movies, err := SearchIMDB(query)
	if err != nil {
		log.Fatal(err)
	}

	// user selects a title
	selectedIMDB, err := FuzzySelect(movies, func(item IMDBItem) string {
		return fmt.Sprintf("%s (%d)", item.Title, item.Year)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("selected imdb", selectedIMDB.ID)

	// resolve stream variants
	opts := extractor.ResolveOptions{
		IMDBID: selectedIMDB.ID,
		Type:   extractor.Movie,
	}
	variants, err := opts.ResolveStreamVariants()
	if err != nil {
		log.Fatal(err)
	}

	// user selects quality
	selected, err := FuzzySelect(variants, func(v extractor.StreamVariant) string {
		return extractor.FormatResolutionQuality(v.Resolution)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	OpenInMPV(selected.URL, selectedIMDB)
}
