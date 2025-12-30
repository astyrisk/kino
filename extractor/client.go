package extractor

import (
	"log"
	"net/http"
	"os"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	userAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"
)

var debugMode = os.Getenv("DEBUG") == "1"

func logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

func logSuccess(format string, v ...interface{}) {
	log.Printf("[SUCCESS] "+format, v...)
}

func logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

func logDebug(format string, v ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	defer time.Sleep(time.Second)
	r.Header.Set("Accept-Language", "en")
	r.Header.Set("User-Agent", userAgent)
	return e.RoundTripper.RoundTrip(r)
}

var client = &http.Client{
	Timeout:   defaultHTTPTimeout,
	Transport: &customTransport{http.DefaultTransport},
}
