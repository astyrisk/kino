package extractor

import (
	"fmt"
	"imdb/client"
	"io"
	"kino/client"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	vidsrcBaseURL      = "https://vidsrc-embed.ru"
	cloudnestraBaseURL = "https://cloudnestra.com"
	httpsScheme        = "https:"
)

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
	client.LogDebug("Fetching page")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for %q: %w", url, err)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.HttpClient.Do(req)
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
	client.LogDebug("Parsing embed HTML for RCP URL")
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
	client.LogDebug("Extracting ProRCP URL from RCP page")
	re := regexp.MustCompile(`src: '(/prorcp/[^']+)`)
	match := re.FindStringSubmatch(rcpHTML)
	if len(match) < 2 {
		return "", fmt.Errorf("no ProRCP URL found in RCP page")
	}
	return match[1], nil
}
