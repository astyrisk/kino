package client

import (
	"log"
	"net/http"
	"os"
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

const defaultHTTPTimeout = 10 * time.Second

var (
	debugMode = os.Getenv("DEBUG") == "1"
	// HttpClient is the shared HTTP client instance with custom transport
	HttpClient = &http.Client{
		Timeout:   defaultHTTPTimeout,
		Transport: New().Transport,
	}
)

// Note: These logging functions use the standard log package instead of the logger
// to avoid clearing the screen for every debug/info message during extraction.
// The main application flow uses the logger for user-facing messages.

func LogInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func LogSuccess(format string, v ...interface{}) {
	log.Printf("[SUCCESS] "+format, v...)
}

func LogError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

func LogDebug(format string, v ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}
