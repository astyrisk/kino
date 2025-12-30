package playback

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"imdb/format"
	"imdb/menu"
	"imdb/player"
	"imdb/stream"
	"imdb/tracking"
)

func HandleStreaming(client *http.Client, imdbID, cacheSize string) error {
	if !player.IsAvailable() {
		fmt.Println("\nWarning: mpv not found in PATH")
		fmt.Println("Please install mpv to enable streaming playback")
		fmt.Println("On Ubuntu/Debian: sudo apt install mpv")
		fmt.Println("On macOS: brew install mpv")
		fmt.Println("On Windows: choco install mpv")
		return nil
	}
	
	mediaType, season, episode := format.ParseIMDbID(imdbID)
	
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
	
	titleInfo := format.GetTitleInfo(client, imdbID)
	
	for {
		fmt.Printf("\nPlaying %s...\n", stream.FormatVariantDisplay(*selectedVariant))
		
		title := format.GetTitleForPlayer(client, imdbID, mediaType, season, episode)
		
		mpvPlayer, err := player.New()
		if err != nil {
			return fmt.Errorf("failed to create player: %w", err)
		}
		
		mpvPlayer.CacheSize = cacheSize
		mpvPlayer.Title = title
		
		playbackDone := make(chan error, 1)
		startTime := time.Now()
		
		go func() {
			err := mpvPlayer.Play(selectedVariant.URL)
			playbackDone <- err
		}()
		
		time.Sleep(500 * time.Millisecond)
		
		_, _, nextErr := menu.GetNextEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
		_, _, prevErr := menu.GetPreviousEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
		
		action, err := menu.ShowPostPlayMenu(format.GetMediaTypeString(mediaType), nextErr == nil, prevErr == nil)
		if err != nil {
			<-playbackDone
			return nil
		}
		
		switch action {
		case "next":
			newSeason, newEpisode, err := menu.GetNextEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				<-playbackDone
				return nil
			}
			imdbID = fmt.Sprintf("%s/%d-%d", strings.Split(imdbID, "/")[0], newSeason, newEpisode)
			season, episode = newSeason, newEpisode
			variants, err = stream.GetStreamVariants(imdbID, mediaType, season, episode)
			if err != nil {
				<-playbackDone
				return fmt.Errorf("failed to get streaming variants: %w", err)
			}
			selectedVariant, err = selectStreamVariant(variants)
			if err != nil {
				<-playbackDone
				return err
			}
			<-playbackDone
			continue
			
		case "replay":
			<-playbackDone
			
		case "previous":
			newSeason, newEpisode, err := menu.GetPreviousEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
			if err != nil {
				fmt.Printf("\nError: %v\n", err)
				<-playbackDone
				return nil
			}
			imdbID = fmt.Sprintf("%s/%d-%d", strings.Split(imdbID, "/")[0], newSeason, newEpisode)
			season, episode = newSeason, newEpisode
			variants, err = stream.GetStreamVariants(imdbID, mediaType, season, episode)
			if err != nil {
				<-playbackDone
				return fmt.Errorf("failed to get streaming variants: %w", err)
			}
			selectedVariant, err = selectStreamVariant(variants)
			if err != nil {
				<-playbackDone
				return err
			}
			<-playbackDone
			continue
			
		case "change_quality":
			variants, err = stream.GetStreamVariants(imdbID, mediaType, season, episode)
			if err != nil {
				<-playbackDone
				return fmt.Errorf("failed to get streaming variants: %w", err)
			}
			selectedVariant, err = selectStreamVariant(variants)
			if err != nil {
				<-playbackDone
				return err
			}
			<-playbackDone
			continue
			
		case "select":
			<-playbackDone
			return nil
			
		case "quit":
			<-playbackDone
			os.Exit(0)
		}
		
		duration := int(time.Since(startTime).Seconds())
		if duration > 0 {
			tracking.RecordWatch(imdbID, titleInfo.Name, format.GetMediaTypeString(mediaType), season, episode, duration)
		}
	}
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
