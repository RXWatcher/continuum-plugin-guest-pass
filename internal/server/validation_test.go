package server

import (
	"testing"
	"time"
)

func TestParseExpiryAbsoluteAndRelative(t *testing.T) {
	// Past absolute is rejected.
	if _, err := parseExpiry(time.Now().Add(-time.Hour).Format(time.RFC3339), 0); err == nil {
		t.Fatal("past expires_at should fail")
	}
	// Excessive relative is rejected.
	if _, err := parseExpiry("", 24*365+1); err == nil {
		t.Fatal("excessive expires_in_hours should fail")
	}
	// Default falls back to 24h.
	got, err := parseExpiry("", 0)
	if err != nil {
		t.Fatalf("default expiry failed: %v", err)
	}
	if time.Until(got) < 23*time.Hour || time.Until(got) > 25*time.Hour {
		t.Fatalf("default expiry %s should be ~24h from now", got)
	}
	// Absolute pass-through.
	want := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	got, err = parseExpiry(want.Format(time.RFC3339), 0)
	if err != nil {
		t.Fatalf("absolute expiry: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("absolute expiry = %s, want %s", got, want)
	}
}

func TestValidTargetTypeOnlyAllowsPlayableMediaFile(t *testing.T) {
	if !validTargetType("media_file") {
		t.Fatal("media_file should be accepted")
	}
	for _, targetType := range []string{"movie", "episode", "series", "collection", "watch_room", "ebook", "audiobook", ""} {
		if validTargetType(targetType) {
			t.Fatalf("target type %q should be rejected until playback grants support it", targetType)
		}
	}
}

func TestResolutionHeight(t *testing.T) {
	cases := map[string]int{
		"480p":  480,
		"720p":  720,
		"1080p": 1080,
		"4k":    2160,
		"2160p": 2160,
		"":      0,
		"junk":  0,
	}
	for in, want := range cases {
		if got := resolutionHeight(in); got != want {
			t.Errorf("resolutionHeight(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestSplitList(t *testing.T) {
	got := splitList("us, nl\n  fr,,de")
	want := []string{"us", "nl", "fr", "de"}
	if len(got) != len(want) {
		t.Fatalf("splitList len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("splitList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCatalogMediaTypes(t *testing.T) {
	if got := catalogMediaTypes(""); len(got) != 2 || got[0] != "movie" || got[1] != "episode" {
		t.Fatalf("default = %v, want [movie episode]", got)
	}
	if got := catalogMediaTypes("MOVIE"); len(got) != 1 || got[0] != "movie" {
		t.Fatalf("movie alias = %v, want [movie]", got)
	}
}

func TestShareURLRelativeAndAbsolute(t *testing.T) {
	if got := shareURL("", "abc"); got != "/p/abc" {
		t.Errorf("relative shareURL = %q, want /p/abc", got)
	}
	if got := shareURL("https://example.com/plugin/", "abc"); got != "https://example.com/plugin/p/abc" {
		t.Errorf("absolute shareURL = %q, want https://example.com/plugin/p/abc", got)
	}
	if got := shareURL("https://example.com/plugin", ""); got != "" {
		t.Errorf("empty token shareURL = %q, want empty", got)
	}
}
