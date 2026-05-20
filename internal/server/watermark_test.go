package server

import (
	"strings"
	"testing"

	"github.com/ContinuumApp/continuum-plugin-guest-pass/internal/store"
)

func TestNormalizeWatermarkMode(t *testing.T) {
	cases := map[string]string{
		"":                       "none",   // empty mode + empty profile defaults to none
		"visible":                "visible",
		"BURNED_IN":              "burned_in",
		"burned-in":              "burned_in",
		"burnedin":               "burned_in",
		"visible, forensic":      "visible,forensic",
		"visible+all":            "visible,all",
		"junk":                   "none",
		"forensic,not-a-mode,all": "forensic,all",
	}
	for in, want := range cases {
		if got := normalizeWatermarkMode(in, ""); got != want {
			t.Errorf("normalizeWatermarkMode(%q,\"\") = %q, want %q", in, got, want)
		}
	}
	// Empty mode with profile set picks "all".
	if got := normalizeWatermarkMode("", "Watermark template"); got != "all" {
		t.Errorf("empty mode + profile = %q, want all", got)
	}
}

func TestRenderWatermarkTextSubstitutesPlaceholders(t *testing.T) {
	p := &store.Pass{ID: 7, Title: "Movie"}
	req := accessRequest{DeviceID: "dev-1"}

	out := renderWatermarkText("Pass {{pass_id}} title {{title}} ip {{ip}} dev {{device_id}}", p, req, "10.0.0.1")
	for _, want := range []string{"Pass 7", "title Movie", "ip 10.0.0.1", "dev dev-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("output %q missing %q", out, want)
		}
	}
}

func TestRenderWatermarkTextDefaultWhenEmpty(t *testing.T) {
	p := &store.Pass{ID: 7}
	got := renderWatermarkText("", p, accessRequest{}, "10.0.0.1")
	if !strings.Contains(got, "Guest pass 7") {
		t.Errorf("default template should include pass id, got %q", got)
	}
}

func TestRenderWatermarkTextEmptyWhenLogoConfigured(t *testing.T) {
	p := &store.Pass{ID: 7, WatermarkLogoURL: "/logo.png"}
	if got := renderWatermarkText("", p, accessRequest{}, "10.0.0.1"); got != "" {
		t.Errorf("with logo and no template, text should be empty; got %q", got)
	}
}

func TestRenderWatermarkTextTruncatesAt160(t *testing.T) {
	p := &store.Pass{ID: 1, Title: strings.Repeat("x", 500)}
	got := renderWatermarkText("title:{{title}}", p, accessRequest{}, "ip")
	if len(got) > 160 {
		t.Errorf("watermark text len = %d, want <=160", len(got))
	}
}
