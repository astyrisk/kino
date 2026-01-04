package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"imdb/client"
	"imdb/playback"
	"imdb/tui"

	"github.com/StalkR/imdb"
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
	log := tui.New()

	for {
		selectedTitle, err := tui.Interactive(httpClient, log)
		if err != nil {
			if err.Error() == "exit" {
				return
			}
			log.Error(err.Error())
			continue
		}

		finalID, err := tui.HandleTitleSelection(httpClient, selectedTitle, log)
		if err != nil {
			if err.Error() == "abort" {
				log.ShowInfo("Selection cancelled.")
				continue
			}
			log.Error(err.Error())
			continue
		}

		log.ShowStatus(fmt.Sprintf("Selected IMDb ID: %s", finalID))

		err = playback.HandleStreaming(httpClient, finalID, *cacheSize, log)
		if err != nil {
			if err.Error() == "abort" {
				log.ShowInfo("Streaming cancelled.")
				continue
			}
			log.Error(err.Error())
			continue
		}

		log.ClearScreen()
	}
}

func runSingleSearchMode(httpClient *http.Client, query string) {
	log := tui.New()

	results, err := imdb.SearchTitle(httpClient, query)
	if err != nil {
		log.Error(err.Error())
		os.Exit(2)
	}
	if len(results) == 0 {
		log.Error("No results found.")
		os.Exit(3)
	}

	selectedTitle, err := tui.SelectTitle(results)
	if err != nil {
		log.Error(err.Error())
		os.Exit(4)
	}

	finalID, err := tui.HandleTitleSelection(httpClient, selectedTitle, log)
	if err != nil {
		log.Error(err.Error())
		os.Exit(5)
	}

	log.ShowStatus(fmt.Sprintf("Selected IMDb ID: %s", finalID))

	err = playback.HandleStreaming(httpClient, finalID, *cacheSize, log)
	if err != nil {
		log.Error(err.Error())
		os.Exit(6)
	}
}
