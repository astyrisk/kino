package ui

import (
	"fmt"
	"strconv"
	"strings"

	"kino/extractor"
)

// FormatVariantDisplay formats a stream variant for display in the UI.
func FormatVariantDisplay(v extractor.StreamVariant) string {
	resolution := FormatResolution(v.Resolution)
	bandwidth := FormatBandwidth(v.Bandwidth)

	if bandwidth != "" {
		return fmt.Sprintf("%s (%s)", resolution, bandwidth)
	}
	return resolution
}

// FormatResolution converts raw resolution strings (e.g., "1920x1080") into friendly names (e.g., "1080p").
func FormatResolution(resolution string) string {
	if !strings.Contains(resolution, "x") {
		return resolution
	}

	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return resolution
	}

	height := parts[1]
	switch height {
	case "1080":
		return "1080p"
	case "720":
		return "720p"
	case "480":
		return "480p"
	case "360":
		return "360p"
	default:
		return resolution
	}
}

// FormatBandwidth converts raw bandwidth in bps to human-readable Mbps or Gbps.
func FormatBandwidth(bandwidth string) string {
	if bandwidth == "" {
		return ""
	}

	if len(bandwidth) > 6 {
		mbps := bandwidth[:len(bandwidth)-6] + "." + bandwidth[len(bandwidth)-6:len(bandwidth)-5]
		if mbpsInt, err := strconv.Atoi(bandwidth[:len(bandwidth)-6]); err == nil {
			if mbpsInt >= 1000 {
				return strconv.Itoa(mbpsInt/1000) + "." + bandwidth[len(bandwidth)-9:len(bandwidth)-8] + " Gbps"
			}
		}
		return mbps + " Mbps"
	}
	return bandwidth + " bps"
}
