package stream

import (
	"fmt"
	"log"

	"kino/extractor"
)

type (
	MediaType      = extractor.MediaType
	ResolveOptions = extractor.ResolveOptions
	StreamVariant  = extractor.StreamVariant
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
