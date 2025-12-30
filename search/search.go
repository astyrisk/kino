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
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func Interactive(client *http.Client) (*imdb.Title, error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Search kino: ")
		
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				return nil, fmt.Errorf("exit")
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

		selectedTitle, err := SelectTitle(results)
		if err != nil {
			if err.Error() == "abort" {
				fmt.Println("Search cancelled.")
				fmt.Println()
				continue
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
