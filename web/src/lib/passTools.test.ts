import { describe, expect, it } from "vitest";
import { buildPolicySummary, duplicatePassForm, eventTone, passTemplates, type PassDraft } from "./passTools";

const basePass: PassDraft = {
  title: "Press screener",
  note: "For review only",
  expires_in_hours: 24,
  valid_hours_after_first_open: 0,
  max_opens: 0,
  max_plays: 1,
  max_watch_minutes: 180,
  max_concurrent_streams: 1,
  max_devices: 1,
  max_resolution: "1080p",
  allow_downloads: false,
  allow_direct_play: false,
  lock_to_first_ip: false,
  require_pin: true,
  pin: "",
  disable_seeking: true,
  watermark_mode: "visible",
  watermark_profile: "Guest pass {{pass_id}}",
  watermark_logo_url: "",
  ip_allowlist: "",
  country_allowlist: "US, NL",
  session_grace_minutes: 0,
  per_item_play_count: false,
  geofence: "",
};

describe("passTools", () => {
  it("provides operational templates for common guest-pass workflows", () => {
    expect(passTemplates.map((template) => template.id)).toEqual([
      "preview-24h",
      "one-time-screening",
      "press-screener",
      "family-share",
    ]);
    expect(passTemplates.find((template) => template.id === "press-screener")?.values.require_pin).toBe(true);
  });

  it("summarizes the effective access policy before creation", () => {
    expect(buildPolicySummary(basePass)).toEqual([
      "Expires in 24 hours",
      "Valid until calendar expiry",
      "1 play",
      "180 watch minutes",
      "1 concurrent stream",
      "1 device",
      "PIN required",
      "Seeking disabled",
      "Visible watermark",
      "Allowed countries: US, NL",
    ]);
  });

  it("copies an existing pass into a safe editable draft without copying secret state", () => {
    const draft = duplicatePassForm({
      ...basePass,
      title: "Original",
      expires_in_hours: 1,
      revoked_at: "2026-05-18T10:00:00Z",
      target_id: "123",
    });

    expect(draft.title).toBe("Copy of Original");
    expect(draft.target_id).toBe("123");
    expect(draft.pin).toBe("");
    expect(draft.expires_in_hours).toBe(24);
    expect(draft.note).toBe("For review only");
  });

  it("classifies denied and successful activity events", () => {
    expect(eventTone("opened")).toBe("success");
    expect(eventTone("grant_minted")).toBe("success");
    expect(eventTone("play_rejected_bad_pin")).toBe("danger");
    expect(eventTone("rejected_expired")).toBe("danger");
    expect(eventTone("created")).toBe("neutral");
  });
});
