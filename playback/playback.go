package playback

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"imdb/extractor"
	"imdb/tui"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

// ===== Type Aliases =====

type (
	MediaType      = extractor.MediaType
	ResolveOptions = extractor.ResolveOptions
	StreamVariant  = extractor.StreamVariant
)

const (
	Movie MediaType = extractor.Movie
	TV    MediaType = extractor.TV
)

// ===== Player Implementation =====

type Player struct {
	playerPath string
	CacheSize  string
	Title      string
	cmd        *exec.Cmd
	mu         sync.Mutex
}

func NewPlayer() (*Player, error) {
	playerPath, err := exec.LookPath("mpv")
	if err != nil {
		return nil, fmt.Errorf("mpv not found in PATH: %w", err)
	}

	return &Player{
		playerPath: playerPath,
	}, nil
}

func (p *Player) Play(url string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	args := []string{}

	if p.CacheSize != "" {
		args = append(args, fmt.Sprintf("--demuxer-max-bytes=%s", p.CacheSize))
	}

	if p.Title != "" {
		args = append(args, fmt.Sprintf("--title=%s", p.Title))
		args = append(args, fmt.Sprintf("--force-media-title=%s", p.Title))
	}

	args = append(args, url)

	p.cmd = exec.Command(p.playerPath, args...)
	p.cmd.Stdin = os.Stdin

	return p.cmd.Run()
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
}

func IsPlayerAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

func PlayURL(url string) error {
	player, err := NewPlayer()
	if err != nil {
		return err
	}

	return player.Play(url)
}

// ===== Stream Variant Functions =====

func GetStreamVariants(imdbID string, mediaType MediaType, season, episode int) ([]StreamVariant, error) {
	opts := ResolveOptions{IMDBID: imdbID, Type: mediaType, Season: season, Episode: episode}

	variants, err := opts.ResolveStreamVariants()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stream variants: %w", err)
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("no streaming variants found")
	}

	return variants, nil
}

func FormatVariantDisplay(v StreamVariant) string {
	resolution := formatResolution(v.Resolution)
	bandwidth := formatBandwidth(v.Bandwidth)

	if bandwidth != "" {
		return fmt.Sprintf("%s (%s)", resolution, bandwidth)
	}
	return resolution
}

func formatResolution(resolution string) string {
	if !strings.Contains(resolution, "x") {
		return resolution
	}

	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return resolution
	}

	height := parts[1]
	switch height {
	case "1080":
		return "1080p"
	case "720":
		return "720p"
	case "480":
		return "480p"
	case "360":
		return "360p"
	default:
		return resolution
	}
}

func formatBandwidth(bandwidth string) string {
	if bandwidth == "" {
		return ""
	}

	if len(bandwidth) > 6 {
		mbps := bandwidth[:len(bandwidth)-6] + "." + bandwidth[len(bandwidth)-6:len(bandwidth)-5]
		if mbpsInt, err := strconv.Atoi(bandwidth[:len(bandwidth)-6]); err == nil {
			if mbpsInt >= 1000 {
				return strconv.Itoa(mbpsInt/1000) + "." + bandwidth[len(bandwidth)-9:len(bandwidth)-8] + " Gbps"
			}
		}
		return mbps + " Mbps"
	}
	return bandwidth + " bps"
}

// ===== Cache Implementation =====

type StreamCache struct {
	mu       sync.RWMutex
	variants map[string][]StreamVariant
}

var streamCache = &StreamCache{
	variants: make(map[string][]StreamVariant),
}

func GetCachedVariants(imdbID string) ([]StreamVariant, bool) {
	streamCache.mu.RLock()
	defer streamCache.mu.RUnlock()

	variants, exists := streamCache.variants[imdbID]
	return variants, exists
}

func SetCachedVariants(imdbID string, variants []StreamVariant) {
	streamCache.mu.Lock()
	defer streamCache.mu.Unlock()

	streamCache.variants[imdbID] = variants
}

func ClearCachedVariants(imdbID string) {
	streamCache.mu.Lock()
	defer streamCache.mu.Unlock()

	delete(streamCache.variants, imdbID)
}

// ===== Main Playback Logic =====

func HandleStreaming(client *http.Client, imdbID, cacheSize string, log *tui.Logger) error {
	if !IsPlayerAvailable() {
		log.ShowInfo("Warning: mpv not found in PATH\nPlease install mpv to enable streaming playback\nOn Ubuntu/Debian: sudo apt install mpv\nOn macOS: brew install mpv\nOn Windows: choco install mpv")
		return nil
	}

	mediaType, season, episode := tui.ParseIMDbID(imdbID)

	log.ShowStatus("Fetching streaming options...")

	var variants []StreamVariant
	var err error

	// Check if variants are cached
	if cachedVariants, exists := GetCachedVariants(imdbID); exists {
		log.ShowMessage("Using cached stream variants...")
		variants = cachedVariants
	} else {
		// Fetch and cache variants
		variants, err = GetStreamVariants(imdbID, mediaType, season, episode)
		if err != nil {
			return fmt.Errorf("failed to get streaming variants: %w", err)
		}

		if len(variants) == 0 {
			return fmt.Errorf("no streaming variants found")
		}

		// Cache the variants
		SetCachedVariants(imdbID, variants)
	}

	selectedVariant, err := selectStreamVariant(variants)
	if err != nil {
		return err
	}

	titleInfo := tui.GetTitleInfo(client, imdbID)

	for {
		log.ShowStatus(fmt.Sprintf("Playing %s...", FormatVariantDisplay(*selectedVariant)))

		title := tui.GetTitleForPlayer(client, imdbID, mediaType, season, episode)

		mpvPlayer, err := NewPlayer()
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

		_, _, nextErr := tui.GetNextEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
		_, _, prevErr := tui.GetPreviousEpisode(client, strings.Split(imdbID, "/")[0], season, episode)

		action, err := tui.ShowPostPlayMenu(tui.GetMediaTypeString(mediaType), nextErr == nil, prevErr == nil, titleInfo.Name, season, episode)
		if err != nil {
			<-playbackDone
			return nil
		}

		switch action {
		case "next":
			mpvPlayer.Stop()
			newSeason, newEpisode, err := tui.GetNextEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
			if err != nil {
				log.Error(err.Error())
				<-playbackDone
				return nil
			}
			imdbID = fmt.Sprintf("%s/%d-%d", strings.Split(imdbID, "/")[0], newSeason, newEpisode)
			season, episode = newSeason, newEpisode

			// Clear cache for new episode and fetch variants
			ClearCachedVariants(imdbID)
			variants, err = GetStreamVariants(imdbID, mediaType, season, episode)
			if err != nil {
				<-playbackDone
				return fmt.Errorf("failed to get streaming variants: %w", err)
			}
			SetCachedVariants(imdbID, variants)

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
			mpvPlayer.Stop()
			newSeason, newEpisode, err := tui.GetPreviousEpisode(client, strings.Split(imdbID, "/")[0], season, episode)
			if err != nil {
				log.Error(err.Error())
				<-playbackDone
				return nil
			}
			imdbID = fmt.Sprintf("%s/%d-%d", strings.Split(imdbID, "/")[0], newSeason, newEpisode)
			season, episode = newSeason, newEpisode

			// Clear cache for new episode and fetch variants
			ClearCachedVariants(imdbID)
			variants, err = GetStreamVariants(imdbID, mediaType, season, episode)
			if err != nil {
				<-playbackDone
				return fmt.Errorf("failed to get streaming variants: %w", err)
			}
			SetCachedVariants(imdbID, variants)

			selectedVariant, err = selectStreamVariant(variants)
			if err != nil {
				<-playbackDone
				return err
			}
			<-playbackDone
			continue

		case "change_quality":
			mpvPlayer.Stop()
			// Use cached variants instead of re-fetching
			if cachedVariants, exists := GetCachedVariants(imdbID); exists {
				variants = cachedVariants
			} else {
				// Fallback to fetching if not cached
				variants, err = GetStreamVariants(imdbID, mediaType, season, episode)
				if err != nil {
					<-playbackDone
					return fmt.Errorf("failed to get streaming variants: %w", err)
				}
				SetCachedVariants(imdbID, variants)
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
			tui.RecordWatch(imdbID, titleInfo.Name, tui.GetMediaTypeString(mediaType), season, episode, duration)
		}
	}
}

func selectStreamVariant(variants []StreamVariant) (*StreamVariant, error) {
	idx, err := fuzzyfinder.Find(
		variants,
		func(i int) string {
			return FormatVariantDisplay(variants[i])
		},
		fuzzyfinder.WithPromptString("Select stream quality:"),
	)

	if err != nil {
		return nil, err
	}

	return &variants[idx], nil
}
