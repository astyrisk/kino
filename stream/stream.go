package stream

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"imdb/extractor"
)

type (
	MediaType    = extractor.MediaType
	ResolveOptions = extractor.ResolveOptions
	StreamVariant = extractor.StreamVariant
)

const (
	Movie MediaType = extractor.Movie
	TV    MediaType = extractor.TV
)

func GetStreamVariants(imdbID string, mediaType MediaType, season, episode int) ([]StreamVariant, error) {
	opts := ResolveOptions{IMDBID: imdbID, Type: mediaType, Season: season, Episode: episode}
	
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