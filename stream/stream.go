package stream

import (
	"fmt"
	"log"
	"strings"

	"imdb/extractor"
)

// MediaType represents the type of content (movie or tv).
type MediaType = extractor.MediaType

const (
	Movie MediaType = extractor.Movie
	TV    MediaType = extractor.TV
)

// ResolveOptions contains the input parameters for resolving an HLS stream.
type ResolveOptions = extractor.ResolveOptions

// StreamVariant represents one HLS variant (quality level).
type StreamVariant = extractor.StreamVariant

// GetStreamVariants fetches streaming variants for the given IMDb ID and media type.
func GetStreamVariants(imdbID string, mediaType MediaType, season, episode int) ([]StreamVariant, error) {
	opts := ResolveOptions{
		IMDBID:  imdbID,
		Type:    mediaType,
		Season:  season,
		Episode: episode,
	}

	log.Printf("Fetching streaming variants for %s (%s)...", imdbID, mediaType)
	variants, err := opts.ResolveStreamVariants()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stream variants: %w", err)
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("no streaming variants found")
	}

	return variants, nil
}

// FormatVariantDisplay formats a stream variant for display in the fuzzy finder.
func FormatVariantDisplay(v StreamVariant) string {
	resolution := formatResolutionQuality(v.Resolution)
	bandwidth := formatBandwidth(v.Bandwidth)
	
	quality := resolution
	if bandwidth != "" {
		quality = fmt.Sprintf("%s (%s)", resolution, bandwidth)
	}
	
	return quality
}

// formatResolutionQuality converts resolution from "1920x1080" format to "1080p"
func formatResolutionQuality(resolution string) string {
	if !strings.Contains(resolution, "x") {
		return resolution
	}
	
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return resolution
	}
	
	height := parts[1]
	qualityMap := map[string]string{
		"1080": "1080p",
		"720":  "720p",
		"480":  "480p",
		"360":  "360p",
	}
	
	if quality, exists := qualityMap[height]; exists {
		return quality
	}
	return resolution
}

// formatBandwidth converts bandwidth from "5000000" to "5.0 Mbps"
func formatBandwidth(bandwidth string) string {
	if bandwidth == "" {
		return ""
	}
	
	if len(bandwidth) > 6 {
		return bandwidth[:len(bandwidth)-6] + "." + bandwidth[len(bandwidth)-6:len(bandwidth)-5] + " Mbps"
	}
	return bandwidth + " bps"
}