package playback

import (
	"imdb/stream"
	"sync"
)

type StreamCache struct {
	mu       sync.RWMutex
	variants map[string][]stream.StreamVariant
}

var streamCache = &StreamCache{
	variants: make(map[string][]stream.StreamVariant),
}

func GetCachedVariants(imdbID string) ([]stream.StreamVariant, bool) {
	streamCache.mu.RLock()
	defer streamCache.mu.RUnlock()
	
	variants, exists := streamCache.variants[imdbID]
	return variants, exists
}

func SetCachedVariants(imdbID string, variants []stream.StreamVariant) {
	streamCache.mu.Lock()
	defer streamCache.mu.Unlock()
	
	streamCache.variants[imdbID] = variants
}

func ClearCachedVariants(imdbID string) {
	streamCache.mu.Lock()
	defer streamCache.mu.Unlock()
	
	delete(streamCache.variants, imdbID)
}