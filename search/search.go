package search

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/StalkR/imdb"
	"imdb/logger"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func Interactive(client *http.Client, log *logger.Logger) (*imdb.Title, error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)

	for {
		// Show search prompt at bottom
		log.ShowPrompt("Search kino: ")
		
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				return nil, fmt.Errorf("exit")
			}
			log.Error(fmt.Sprintf("Error reading input: %v", err))
			continue
		}

		query := strings.TrimSpace(input)
		if query == "" {
			continue
		}

		// Show searching status
		log.ShowStatus(fmt.Sprintf("Searching for: %s...", query))

		results, err := imdb.SearchTitle(client, query)
		if err != nil {
			log.Error(fmt.Sprintf("Error searching: %v", err))
			continue
		}

		if len(results) == 0 {
			log.ShowInfo("No results found.")
			continue
		}

		selectedTitle, err := SelectTitle(results)
		if err != nil {
			if err.Error() == "abort" {
				log.ShowInfo("Search cancelled.")
				continue
			}
			log.Error(err.Error())
			continue
		}

		return selectedTitle, nil
	}
}

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
