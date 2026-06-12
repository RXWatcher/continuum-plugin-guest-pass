package server

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// validTargetType only allows media_file targets; other types lack
// playback grant support.
func validTargetType(v string) bool {
	return v == "media_file"
}

func validAbsoluteURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// parseExpiry resolves an absolute (RFC3339) or relative (hours-from-now)
// expiry. Falls back to 24 hours when both are zero. Caps at one year out
// to prevent obvious typos creating effectively-permanent shares.
func parseExpiry(raw string, hours int) (time.Time, error) {
	now := time.Now()
	if raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, fmt.Errorf("expires_at must be RFC3339")
		}
		if !t.After(now) {
			return time.Time{}, fmt.Errorf("expires_at must be in the future")
		}
		if t.After(now.Add(24 * 365 * time.Hour)) {
			return time.Time{}, fmt.Errorf("expires_at cannot exceed one year from now")
		}
		return t, nil
	}
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*365 {
		return time.Time{}, fmt.Errorf("expires_in_hours cannot exceed one year")
	}
	return now.Add(time.Duration(hours) * time.Hour), nil
}

func catalogMediaTypes(raw string) []string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "movie":
		return []string{"movie"}
	case "episode":
		return []string{"episode"}
	case "series":
		return []string{"series"}
	default:
		return []string{"movie", "episode"}
	}
}

func parseBoundedInt(raw string, fallback, minValue, maxValue int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		value = fallback
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func resolutionHeight(value string) int {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "480p":
		return 480
	case "720p":
		return 720
	case "1080p":
		return 1080
	case "4k", "2160p":
		return 2160
	default:
		return 0
	}
}

func nonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func defaultPositive(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func splitList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
