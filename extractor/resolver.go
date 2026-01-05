package extractor

import (
	"fmt"
	"imdb/client"
	"io"
	"kino/client"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	contentDirectory       = "content"
	counterFileName        = "content/counter.txt"
	placeholderReplacement = "cloudnestra.com"
	defaultStartCounter    = 1
)

var fileCounter int

type MediaType string

const (
	Movie MediaType = "movie"
	TV    MediaType = "tv"
)

type ResolveOptions struct {
	IMDBID  string
	Type    MediaType
	Season  int
	Episode int
}

type StreamVariant struct {
	Resolution string
	Bandwidth  string
	URL        string
}

func (opts ResolveOptions) ResolveVariants() (string, error) {
	mediaType := "movie"
	if opts.Type == TV {
		mediaType = "TV show"
	}
	client.LogInfo("Resolving stream for %s (%s)...", mediaType, opts.IMDBID)

	embedURL, err := opts.constructEmbedURL()
	if err != nil {
		return "", err
	}
	client.LogDebug("Built embed URL")

	embedHTML, err := fetchContent(embedURL, "")
	if err != nil {
		return "", err
	}

	rcpURL, err := extractRCPURL(embedHTML)
	if err != nil {
		return "", err
	}
	client.LogSuccess("Extracted RCP URL")

	rcpHTML, err := fetchContent(httpsScheme+rcpURL, "")
	if err != nil {
		return "", err
	}

	proRCPURL, err := extractProRCPURL(rcpHTML)
	if err != nil {
		return "", err
	}
	client.LogSuccess("Extracted ProRCP URL")

	proRCPHTML, err := fetchContent(cloudnestraBaseURL+proRCPURL, cloudnestraBaseURL)
	if err != nil {
		return "", err
	}

	decodedURL, err := decodeStreamURL(proRCPHTML)
	if err != nil {
		return "", err
	}
	client.LogSuccess("Decoded stream URL")

	decodedArr := processAndDeduplicateStreamURLs(decodedURL)

	for _, testURL := range decodedArr {
		client.LogDebug("Testing URL viability")
		parsedURL, err := url.Parse(testURL)
		if err != nil {
			client.LogDebug("Failed to parse URL: %v", err)
			continue
		}
		resp, err := client.HttpClient.Get(parsedURL.String())
		if err != nil {
			client.LogDebug("Failed to fetch: %v", err)
			continue
		}
		client.LogDebug("Response status: %d %s", resp.StatusCode, resp.Status)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return parsedURL.String(), nil
		}
	}

	return "", fmt.Errorf("no successful URL found in %d decoded URLs", len(decodedArr))
}

func processAndDeduplicateStreamURLs(decodedURL string) []string {
	streamURLs := strings.Split(decodedURL, "or")
	client.LogDebug("Processing %d stream URLs", len(streamURLs))

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

		if !seenURLs[trimmedURL] {
			seenURLs[trimmedURL] = true
			uniqueURLs = append(uniqueURLs, trimmedURL)
		}
	}

	client.LogDebug("Filtered to %d unique URLs", len(uniqueURLs))
	return uniqueURLs
}

func (o ResolveOptions) ResolveStreamVariants() ([]StreamVariant, error) {
	masterURL, err := o.ResolveVariants()
	if err != nil {
		return nil, err
	}
	client.LogDebug("Fetching master playlist")

	resp, err := client.HttpClient.Get(masterURL)
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
					client.LogDebug("Found variant: %s, %s", resolution, bandwidth)
				}
			}
		}
	}

	if len(variants) == 0 {
		return nil, fmt.Errorf("no stream variants found in master playlist %q", masterURL)
	}

	client.LogSuccess("Found %d stream variant(s)", len(variants))
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

func decodeStreamURL(proRCPHTML string) (string, error) {
	client.LogInfo("Decoding obfuscated stream...")

	fileCounter = readCounter()
	client.LogDebug("File counter: %d", fileCounter)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(proRCPHTML))
	if err != nil {
		return "", fmt.Errorf("parsing ProRCP HTML: %w", err)
	}

	divSel := doc.Find("div[style='display:none;']")
	if divSel.Length() == 0 {
		client.LogError("No hidden div found")
		return "", fmt.Errorf("no hidden div found")
	}

	encodedString := strings.TrimSpace(divSel.First().Text())
	divId, _ := divSel.Attr("id")
	divHtml, _ := divSel.Html()

	client.LogDebug("Extracted encoded string (%d chars)", len(encodedString))
	client.LogDebug("Div ID: %s", divId)

	if encodedString != "" {
		client.LogDebug("Attempting decode")
		decodedURL, err := DecodeString(encodedString)
		if err == nil && decodedURL != "" {
			return decodedURL, nil
		}
		client.LogError("Decode failed: %v", err)
	}

	client.LogInfo("Saving files for manual inspection")

	scriptSel := doc.Find("script[src*='/sV05kUlNvOdOxvtC/']")
	if scriptSel.Length() > 0 {
		src, exists := scriptSel.First().Attr("src")
		if exists {
			fullURL := cloudnestraBaseURL + src
			client.LogDebug("JS file URL: %s", fullURL)

			jsContent, err := fetchContent(fullURL, cloudnestraBaseURL)
			if err != nil {
				client.LogDebug("Failed to fetch JS content: %v", err)
			} else {
				if err := os.MkdirAll(contentDirectory, 0755); err != nil {
					client.LogDebug("Failed to create content directory: %v", err)
				} else {
					scriptPath := fmt.Sprintf("%s/%d.js", contentDirectory, fileCounter)
					if err := os.WriteFile(scriptPath, []byte(jsContent), 0644); err != nil {
						client.LogDebug("Failed to write JS file: %v", err)
					} else {
						client.LogDebug("Saved JS: %s", scriptPath)
					}
				}
			}
		}
	} else {
		client.LogDebug("No script found with expected pattern")
	}

	htmlPath := fmt.Sprintf("%s/%d.html", contentDirectory, fileCounter)
	htmlContent := fmt.Sprintf("<!DOCTYPE html>\n<html>\n\n<div id=\"%s\" style=\"display: none;\">\n%s\n</div>\n\n<div id=\"output\"></div>\n\n<script src=\"%d.js\"></script>\n\n<script>\ntry {\n    const decodedUrl = window.%s;\n    document.getElementById('output').innerText = 'Decoded URL: ' + decodedUrl;\n} catch (e) {\n    document.getElementById('output').innerText = 'Error decoding URL: ' + e.message;\n}\n</script>\n\n</html>", divId, divHtml, fileCounter, divId)
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		client.LogDebug("Failed to write HTML file: %v", err)
	} else {
		client.LogDebug("Saved HTML: %s", htmlPath)
	}

	fileCounter++
	writeCounter(fileCounter)

	client.LogInfo("Files saved in %s/ directory", contentDirectory)
	return "", nil
}

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

func writeCounter(counter int) {
	os.MkdirAll(contentDirectory, 0755)
	os.WriteFile(counterFileName, []byte(strconv.Itoa(counter)), 0644)
}

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

func formatBandwidth(bandwidth string) string {
	if bandwidth == "" {
		return ""
	}

	if len(bandwidth) > 6 {
		return bandwidth[:len(bandwidth)-6] + "." + bandwidth[len(bandwidth)-6:len(bandwidth)-5] + " Mbps"
	}
	return bandwidth + " bps"
}

func printStreamVariants(variants []StreamVariant) {
	client.LogSuccess("Resolved %d stream variant(s):", len(variants))
	for i, variant := range variants {
		resolution := formatResolutionQuality(variant.Resolution)
		bandwidth := formatBandwidth(variant.Bandwidth)

		log.Printf("  [%d] %s (%s) - %s", i, resolution, bandwidth, variant.URL)
	}
}
