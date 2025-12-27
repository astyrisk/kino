package extractor

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	vidsrcBaseURL        = "https://vidsrc-embed.ru"
	cloudnestraBaseURL   = "https://cloudnestra.com"
	defaultHTTPTimeout   = 10 * time.Second
	contentDirectory     = "content"
	counterFileName      = "content/counter.txt"
	placeholderReplacement = "cloudnestra.com"
	httpsScheme          = "https:"
	defaultStartCounter  = 1
)

// debugMode controls whether debug logs are shown
var debugMode = os.Getenv("DEBUG") == "1"

// logInfo prints informational messages
func logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

// logSuccess prints success messages
func logSuccess(format string, v ...interface{}) {
	log.Printf("[SUCCESS] "+format, v...)
}

// logError prints error messages
func logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// logDebug prints debug messages (only shown if DEBUG=1)
func logDebug(format string, v ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, v...)
	}
}

var client = &http.Client{
	Timeout:   defaultHTTPTimeout,
	Transport: &customTransport{http.DefaultTransport},
}

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"

type customTransport struct {
	http.RoundTripper
}

func (e *customTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	defer time.Sleep(time.Second)         // don't go too fast or risk being blocked by awswaf
	r.Header.Set("Accept-Language", "en") // avoid IP-based language detection
	r.Header.Set("User-Agent", userAgent)
	return e.RoundTripper.RoundTrip(r)
}

var fileCounter int

// MediaType is the type of content (movie or tv).
type MediaType string

const (
	Movie MediaType = "movie"
	TV    MediaType = "tv"
)

// ResolveOptions contains the input parameters for resolving an HLS stream.
type ResolveOptions struct {
	IMDBID  string
	Type    MediaType
	Season  int
	Episode int
}

// StreamVariant represents one HLS variant (quality level).
type StreamVariant struct {
	Resolution string
	Bandwidth  string
	URL        string
}

// ResolveVariants runs the full resolution pipeline and returns the final HLS master URL.
func (opts ResolveOptions) ResolveVariants() (string, error) {
	mediaType := "movie"
	if opts.Type == TV {
		mediaType = "TV show"
	}
	logInfo("Resolving stream for %s (%s)...", mediaType, opts.IMDBID)

	// Step 1: Build and fetch the initial embed page
	embedURL, err := opts.constructEmbedURL()
	if err != nil {
		return "", err
	}
	logDebug("Built embed URL")

	embedHTML, err := fetchContent(embedURL, "")
	if err != nil {
		return "", err
	}

	// Step 2: Extract the RCP URL from the iframe
	rcpURL, err := extractRCPURL(embedHTML)
	if err != nil {
		return "", err
	}
	logSuccess("Extracted RCP URL")

	// Step 3: Fetch the RCP page content
	rcpHTML, err := fetchContent(httpsScheme+rcpURL, "")
	if err != nil {
		return "", err
	}

	// Step 4: Extract the ProRCP URL from the RCP page
	proRCPURL, err := extractProRCPURL(rcpHTML)
	if err != nil {
		return "", err
	}
	logSuccess("Extracted ProRCP URL")

	// Step 5: Fetch the ProRCP page with the correct Referer
	proRCPHTML, err := fetchContent(cloudnestraBaseURL+proRCPURL, cloudnestraBaseURL)
	if err != nil {
		return "", err
	}

	// Step 6: Try to decode, otherwise save
	decodedURL, err := decodeStreamURL(proRCPHTML)
	if err != nil {
		return "", err
	}
	logSuccess("Decoded stream URL")

	decodedArr := processAndDeduplicateStreamURLs(decodedURL)

	// Try each URL and return the first successful one
	for _, testURL := range decodedArr {
		logDebug("Testing URL viability")
		parsedURL, err := url.Parse(testURL)
		if err != nil {
			logDebug("Failed to parse URL: %v", err)
			continue
		}
		resp, err := client.Get(parsedURL.String())
		if err != nil {
			logDebug("Failed to fetch: %v", err)
			continue
		}
		logDebug("Response status: %d %s", resp.StatusCode, resp.Status)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return parsedURL.String(), nil
		}
	}

	return "", fmt.Errorf("no successful URL found in %d decoded URLs", len(decodedArr))
}

// processAndDeduplicateStreamURLs processes a decoded URL string by splitting it,
// replacing placeholders, and filtering out duplicate stream URLs
func processAndDeduplicateStreamURLs(decodedURL string) []string {
	// Split the decoded string with "or" to get individual stream URLs
	streamURLs := strings.Split(decodedURL, "or")
	logDebug("Processing %d stream URLs", len(streamURLs))

	// Replace placeholders and filter out duplicates
	uniqueURLs := make([]string, 0)
	seenURLs := make(map[string]bool)

	replacer := strings.NewReplacer(
		"{v1}", placeholderReplacement,
		"{v2}", placeholderReplacement,
		"{v3}", placeholderReplacement,
		"{v4}", placeholderReplacement,
	)

	for _, urlPart := range streamURLs {
		replacedURL := replacer.Replace(urlPart)
		trimmedURL := strings.TrimSpace(replacedURL)

		// Filter out duplicate URLs
		if !seenURLs[trimmedURL] {
			seenURLs[trimmedURL] = true
			uniqueURLs = append(uniqueURLs, trimmedURL)
		}
	}

	logDebug("Filtered to %d unique URLs", len(uniqueURLs))
	return uniqueURLs
}
func (o ResolveOptions) ResolveStreamVariants() ([]StreamVariant, error) {
	masterURL, err := o.ResolveVariants()
	if err != nil {
		return nil, err
	}
	logDebug("Fetching master playlist")

	resp, err := client.Get(masterURL)
	if err != nil {
		return nil, fmt.Errorf("fetching master playlist %q: %w", masterURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for master playlist %q", resp.StatusCode, masterURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading master playlist %q: %w", masterURL, err)
	}

	lines := strings.Split(string(body), "\n")
	var variants []StreamVariant

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#EXT-X-STREAM-INF") {
			attrs := parseAttributes(line)
			resolution := attrs["RESOLUTION"]
			bandwidth := attrs["BANDWIDTH"]
			if i+1 < len(lines) {
				urlLine := strings.TrimSpace(lines[i+1])
				if urlLine != "" && !strings.HasPrefix(urlLine, "#") {
					abs := resolveRelativeURL(masterURL, urlLine)
					variant := StreamVariant{
						Resolution: resolution,
						Bandwidth:  bandwidth,
						URL:        abs,
					}
					variants = append(variants, variant)
					logDebug("Found variant: %s, %s", resolution, bandwidth)
				}
			}
		}
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("no stream variants found in master playlist %q", masterURL)
	}

	logSuccess("Found %d stream variant(s)", len(variants))
	return variants, nil
}

func parseAttributes(line string) map[string]string {
	attrs := map[string]string{}
	parts := strings.Split(line, ",")
	for _, part := range parts {
		if strings.Contains(part, "=") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			key := kv[0]
			val := strings.Trim(kv[1], "\"")
			attrs[key] = val
		}
	}
	return attrs
}

func resolveRelativeURL(baseStr, refStr string) string {
	base, err := url.Parse(baseStr)
	if err != nil {
		return refStr
	}
	ref, err := url.Parse(refStr)
	if err != nil {
		return refStr
	}
	return base.ResolveReference(ref).String()
}

func (opts ResolveOptions) constructEmbedURL() (string, error) {
	switch opts.Type {
	case Movie:
		if opts.IMDBID == "" {
			return "", fmt.Errorf("cannot build movie URL: imdbId is empty")
		}
		return fmt.Sprintf("%s/embed/movie?imdb=%s", vidsrcBaseURL, opts.IMDBID), nil

	case TV:
		if opts.IMDBID == "" {
			return "", fmt.Errorf("cannot build tv URL: imdbId is empty")
		}
		if opts.Season == 0 || opts.Episode == 0 {
			return "", fmt.Errorf("cannot build tv URL for imdbId %q: season and episode must be set", opts.IMDBID)
		}
		return fmt.Sprintf("%s/embed/tv?imdb=%s&season=%d&episode=%d",
			vidsrcBaseURL, opts.IMDBID, opts.Season, opts.Episode), nil

	default:
		return "", fmt.Errorf("unsupported media type %q for imdbId %q", opts.Type, opts.IMDBID)
	}
}

func fetchContent(url, referer string) (string, error) {
	logDebug("Fetching page")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %q: %w", url, err)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching page %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d for page %q", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading page body %q: %w", url, err)
	}
	return string(body), nil
}

func extractRCPURL(embedHTML string) (string, error) {
	logDebug("Parsing embed HTML for RCP URL")
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(embedHTML))
	if err != nil {
		return "", fmt.Errorf("parsing embed HTML: %w", err)
	}

	src, exists := doc.Find("iframe#player_iframe").Attr("src")
	if !exists || src == "" {
		return "", fmt.Errorf("no iframe src found for RCP URL")
	}
	return src, nil
}

func extractProRCPURL(rcpHTML string) (string, error) {
	logDebug("Extracting ProRCP URL from RCP page")
	re := regexp.MustCompile(`src: '(/prorcp/[^']+)`)
	match := re.FindStringSubmatch(rcpHTML)
	if len(match) < 2 {
		return "", fmt.Errorf("no ProRCP URL found in RCP page")
	}
	return match[1], nil
}

func decodeStreamURL(proRCPHTML string) (string, error) {
	logInfo("Decoding obfuscated stream...")

	// Initialize counter
	fileCounter = readCounter()
	logDebug("File counter: %d", fileCounter)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(proRCPHTML))
	if err != nil {
		return "", fmt.Errorf("parsing ProRCP HTML: %w", err)
	}

	// Extract Hidden Div Content (encoded string)
	divSel := doc.Find("div[style='display:none;']")
	if divSel.Length() == 0 {
		logError("No hidden div found")
		return "", fmt.Errorf("no hidden div found")
	}

	// Get the encoded string from the hidden div
	encodedString := strings.TrimSpace(divSel.First().Text())
	divId, _ := divSel.Attr("id")
	divHtml, _ := divSel.Html()

	logDebug("Extracted encoded string (%d chars)", len(encodedString))
	logDebug("Div ID: %s", divId)

	// Try to decode the encoded string
	if encodedString != "" {
		logDebug("Attempting decode")
		decodedURL, err := DecodeString(encodedString)
		if err == nil && decodedURL != "" {
			// Success! Decoding worked
			return decodedURL, nil
		}
		logError("Decode failed: %v", err)
	}

	// Decoding failed, fall back to saving files for manual inspection
	logInfo("Saving files for manual inspection")

	// 1. Extract and Save JS File
	scriptSel := doc.Find("script[src*='/sV05kUlNvOdOxvtC/']")
	if scriptSel.Length() > 0 {
		src, exists := scriptSel.First().Attr("src")
		if exists {
			fullURL := cloudnestraBaseURL + src
			logDebug("JS file URL: %s", fullURL)

			// Fetch content
			jsContent, err := fetchContent(fullURL, cloudnestraBaseURL)
			if err != nil {
				logDebug("Failed to fetch JS content: %v", err)
			} else {
				// Save to file with counter
				if err := os.MkdirAll(contentDirectory, 0755); err != nil {
					logDebug("Failed to create content directory: %v", err)
				} else {
					scriptPath := fmt.Sprintf("%s/%d.js", contentDirectory, fileCounter)
					if err := os.WriteFile(scriptPath, []byte(jsContent), 0644); err != nil {
						logDebug("Failed to write JS file: %v", err)
					} else {
						logDebug("Saved JS: %s", scriptPath)
					}
				}
			}
		}
	} else {
		logDebug("No script found with expected pattern")
	}

	// 2. Save hidden div content to HTML file
	htmlPath := fmt.Sprintf("%s/%d.html", contentDirectory, fileCounter)
	htmlContent := fmt.Sprintf("<!DOCTYPE html>\n<html>\n\n<div id=\"%s\" style=\"display: none;\">\n%s\n</div>\n\n<div id=\"output\"></div>\n\n<script src=\"%d.js\"></script>\n\n<script>\ntry {\n    const decodedUrl = window.%s;\n    document.getElementById('output').innerText = 'Decoded URL: ' + decodedUrl;\n} catch (e) {\n    document.getElementById('output').innerText = 'Error decoding URL: ' + e.message;\n}\n</script>\n\n</html>", divId, divHtml, fileCounter, divId)
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		logDebug("Failed to write HTML file: %v", err)
	} else {
		logDebug("Saved HTML: %s", htmlPath)
	}

	// Increment and save counter
	fileCounter++
	writeCounter(fileCounter)

	logInfo("Files saved in %s/ directory", contentDirectory)
	return "", nil
}

// readCounter reads the counter from file
func readCounter() int {
	data, err := os.ReadFile(counterFileName)
	if err != nil {
		return defaultStartCounter
	}
	counter, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	if counter == 0 {
		counter = defaultStartCounter
	}
	return counter
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// writeCounter writes the counter to file
func writeCounter(counter int) {
	os.MkdirAll(contentDirectory, 0755)
	os.WriteFile(counterFileName, []byte(strconv.Itoa(counter)), 0644)
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

// printStreamVariants displays the resolved stream variants
func printStreamVariants(variants []StreamVariant) {
	logSuccess("Resolved %d stream variant(s):", len(variants))
	for i, variant := range variants {
		resolution := formatResolutionQuality(variant.Resolution)
		bandwidth := formatBandwidth(variant.Bandwidth)
		
		log.Printf("  [%d] %s (%s) - %s", i, resolution, bandwidth, variant.URL)
	}
}

func main() {
	opts := ResolveOptions{
		IMDBID:  "tt5950044",
		Type:    Movie,
		Season:  0,
		Episode: 0,
	}

	variants, err := opts.ResolveStreamVariants()
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}

	if len(variants) > 0 {
		printStreamVariants(variants)
	} else {
		logInfo("Could not decode URL automatically, files saved for manual inspection")
	}
}
