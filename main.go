package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"kino/client"
	"kino/playback"
	"kino/tui"

	"github.com/gdamore/tcell/v2"
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
	log, err := tui.NewLogger()
	if err != nil {
		fmt.Printf("Error creating logger: %v\n", err)
		return
	}
	defer log.Close()

	for {
		switch ev := log.PollEvent().(type) {

        case *tcell.EventResize:
			log.Sync()
			log.ShowInfo("Resized!")

		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape {
				return
			}

			if ev.Key() == tcell.KeyEnter {
				// log.ShowInfo(fmt.Sprintf("Logged at %s", time.Now().Format("15:04:05")))
				log.ShowStatus(fmt.Sprintf("Logged at %s", time.Now().Format("15:04:05")))
			}
		}
	}
}

// func runInteractiveMode(httpClient *http.Client) {
// 	log, err := tui.NewLogger()

// 	if err != nil {
// 		fmt.Printf("Error creating logger: %v\n", err)
// 		return
// 	}

// 	log.Println("hi")
// 	defer log.Close()

// 	for {
// 		// switch ev := log.PollEvent().(type) {
// 		// switch ev := log.screen.PollEvent().(type) {
// 		switch ev := log.GetScreen().PollEvent().(type) {
// 		case *tcell.EventResize:
// 			s.Sync()
// 			logToBottom("Resized!") // Re-draw at new bottom
// 		case *tcell.EventKey:
// 			if ev.Key() == tcell.KeyEscape {
// 				return
// 			}
// 			if ev.Key() == tcell.KeyEnter {
// 				logToBottom(fmt.Sprintf("Logged at %s", time.Now().Format("15:04:05")))
// 			}
// 		}
// 	}




// 	// for {
// 	// selectedTitle, err := tui.Interactive(httpClient, log)
// 	// if err != nil {
// 	// 	if err.Error() == "exit" {
// 	// 		return
// 	// 	}
// 	// 	log.Error(err.Error())
// 	// 	continue
// 	// }

// 	// fmt.Println("selected title", selectedTitle)

// 	// finalID, err := tui.HandleTitleSelection(httpClient, selectedTitle, log)
// 	// if err != nil {
// 	// 	if err.Error() == "abort" {
// 	// 		log.ShowInfo("Selection cancelled.")
// 	// 		continue
// 	// 	}
// 	// 	log.Error(err.Error())
// 	// 	continue
// 	// }

// 	// log.ShowStatus(fmt.Sprintf("Selected IMDb ID: %s", finalID))

// 	// err = playback.HandleStreaming(httpClient, finalID, *cacheSize, log)
// 	// if err != nil {
// 	// 	if err.Error() == "abort" {
// 	// 		log.ShowInfo("Streaming cancelled.")
// 	// 		continue
// 	// 	}
// 	// 	log.Error(err.Error())
// 	// 	continue
// 	// }

// 	// log.Clear()
// 	// }
// }

func runSingleSearchMode(httpClient *http.Client, query string) {
	log, err := tui.NewLogger()
	if err != nil {
		fmt.Printf("Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	results, err := imdb.SearchTitle(httpClient, query)
	if err != nil {
		log.ShowError(err.Error())
		os.Exit(2)
	}
	if len(results) == 0 {
		log.ShowError("No results found.")
		os.Exit(3)
	}

	selectedTitle, err := tui.SelectTitle(results)
	if err != nil {
		log.ShowError(err.Error())
		os.Exit(4)
	}

	finalID, err := tui.HandleTitleSelection(httpClient, selectedTitle, log)
	if err != nil {
		log.ShowError(err.Error())
		os.Exit(5)
	}

	log.ShowStatus(fmt.Sprintf("Selected IMDb ID: %s", finalID))

	err = playback.HandleStreaming(httpClient, finalID, *cacheSize, log)
	if err != nil {
		log.ShowError(err.Error())
		os.Exit(6)
	}
}
