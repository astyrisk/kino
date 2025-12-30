package client

import (
	"net/http"
	"time"
)

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	defer time.Sleep(time.Second)         // don't go too fast or risk being blocked by awswaf
	r.Header.Set("Accept-Language", "en") // avoid IP-based language detection
	r.Header.Set("User-Agent", UserAgent)
	return e.RoundTripper.RoundTrip(r)
}

func New() *http.Client {
	return &http.Client{
		Transport: &customTransport{http.DefaultTransport},
	}
}
