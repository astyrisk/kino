package client

import (
	"net/http"
	"time"
)

// IMDb deployed awswaf and denies requests using the default Go user-agent (Go-http-client/1.1).
// For now it still allows requests from a browser user-agent. Remain respectful, no spam, etc.
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// don't go too fast or risk being blocked by awswaf
	defer time.Sleep(time.Second)

	// avoid IP-based language detection
	r.Header.Set("Accept-Language", "en")
	r.Header.Set("User-Agent", userAgent)

	return e.RoundTripper.RoundTrip(r)
}

// New returns a new http.Client with custom transport settings.
func New() *http.Client {
	return &http.Client{
		Transport: &customTransport{http.DefaultTransport},
	}
}

// NewWithTimeout returns a new http.Client with custom transport settings and a timeout.
func NewWithTimeout(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &customTransport{http.DefaultTransport},
	}
}
