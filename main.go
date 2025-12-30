package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/StalkR/imdb"
	"imdb/client"
	"imdb/playback"
	"imdb/search"
	"imdb/selection"
)

var cacheSize = flag.String("cache", "12MiB", "Cache size limit for mpv (e.g., 30MiB, 50MiB)")

func main() {
	flag.Parse()
	
	httpClient := client.New()

	if flag.NArg() == 0 {
		runInteractiveMode(httpClient)
		return
	}

	runSingleSearchMode(httpClient, flag.Arg(0))
}

func runInteractiveMode(httpClient *http.Client) {
	for {
		selectedTitle, err := search.Interactive(httpClient)
		if err != nil {
			if err.Error() == "exit" {
				return
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		finalID, err := selection.HandleTitleSelection(httpClient, selectedTitle)
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
		
		err = playback.HandleStreaming(httpClient, finalID, *cacheSize)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("Streaming cancelled.")
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		
		fmt.Println()
		fmt.Println()
	}
}

func runSingleSearchMode(httpClient *http.Client, query string) {
	results, err := imdb.SearchTitle(httpClient, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "No results found.\n")
		os.Exit(3)
	}

	selectedTitle, err := search.SelectTitle(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(4)
	}

	finalID, err := selection.HandleTitleSelection(httpClient, selectedTitle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(5)
	}

	fmt.Printf("\nSelected IMDb ID: %s\n\n", finalID)
	
	err = playback.HandleStreaming(httpClient, finalID, *cacheSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(6)
	}
}
