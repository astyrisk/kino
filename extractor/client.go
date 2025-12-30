package extractor

import (
	"log"
	"net/http"
	"os"
	"time"

	"imdb/client"
)

const defaultHTTPTimeout = 10 * time.Second

var (
	debugMode = os.Getenv("DEBUG") == "1"
	// httpClient is the shared HTTP client instance with custom transport
	httpClient = &http.Client{
		Timeout:   defaultHTTPTimeout,
		Transport: client.New().Transport,
	}
)

// Note: These logging functions use the standard log package instead of the logger
// to avoid clearing the screen for every debug/info message during extraction.
// The main application flow uses the logger for user-facing messages.

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
