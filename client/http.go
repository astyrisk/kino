package client

import (
	"net/http"
	"time"
)

// UserAgent is the browser user-agent used to bypass IMDb's awswaf blocking
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

// customTransport wraps http.RoundTripper to add custom headers and rate limiting
type customTransport struct {
	http.RoundTripper
}

// RoundTrip implements http.RoundTripper interface with custom headers and rate limiting
func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	defer time.Sleep(time.Second)         // don't go too fast or risk being blocked by awswaf
	r.Header.Set("Accept-Language", "en") // avoid IP-based language detection
	r.Header.Set("User-Agent", UserAgent)
	return e.RoundTripper.RoundTrip(r)
}

// New creates a new HTTP client with custom transport
func New() *http.Client {
	return &http.Client{
		Transport: &customTransport{http.DefaultTransport},
	}
}
