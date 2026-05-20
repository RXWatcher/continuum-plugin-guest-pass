package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ContinuumApp/continuum-plugin-guest-pass/internal/store"
)

// normalizeWatermarkMode parses the operator-supplied watermark mode
// string (which may be a comma/space/plus separated list of modes) into a
// canonical comma-joined form. Unknown modes drop silently.
func normalizeWatermarkMode(mode, profile string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		if strings.TrimSpace(profile) == "" {
			return "none"
		}
		return "all"
	}
	valid := map[string]bool{
		"none": true, "visible": true, "burned_in": true, "forensic": true, "all": true,
	}
	parts := strings.FieldsFunc(mode, func(r rune) bool {
		return r == ',' || r == ' ' || r == '+'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ReplaceAll(part, "-", "_")
		if part == "burnedin" {
			part = "burned_in"
		}
		if valid[part] {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ",")
}

// renderWatermarkText substitutes {{placeholder}} tokens in the operator's
// watermark template. The template comes from a trusted source (admin
// config); per-request values (ip, device, time) are inserted but not
// re-substituted, so a malicious title containing "{{pass_id}}" can't
// trigger recursive replacement.
func renderWatermarkText(template string, p *store.Pass, req accessRequest, ip string) string {
	template = strings.TrimSpace(template)
	if template == "" {
		if strings.TrimSpace(p.WatermarkLogoURL) != "" {
			return ""
		}
		template = "Guest pass {{pass_id}} · {{ip}} · {{time}}"
	}
	values := map[string]string{
		"{{pass_id}}":   strconv.Itoa(p.ID),
		"{{title}}":     p.Title,
		"{{ip}}":        ip,
		"{{device_id}}": req.DeviceID,
		"{{subject}}":   fmt.Sprintf("guest-pass:%d", p.ID),
		"{{time}}":      time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range values {
		template = strings.ReplaceAll(template, key, value)
	}
	if len(template) > 160 {
		template = template[:160]
	}
	return template
}
