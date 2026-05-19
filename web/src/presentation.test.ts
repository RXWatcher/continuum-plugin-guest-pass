import { describe, expect, it } from "vitest";
import {
  buildGuestFacts,
  buildPassRowMeta,
  buildPassRowTags,
  formatUsageStat,
} from "./presentation";

describe("buildPassRowMeta", () => {
  it("prefers readable expiry text over raw target ids", () => {
    expect(
      buildPassRowMeta({
        effective_expires_at: "2026-05-20T12:00:00Z",
        max_resolution: "1080p",
        target_type: "media_file",
        target_id: "42",
      }),
    ).toContain("Expires");
  });
});

describe("buildPassRowTags", () => {
  it("surfaces key restrictions only when enabled", () => {
    expect(
      buildPassRowTags({
        require_pin: true,
        lock_to_first_ip: true,
        allow_downloads: false,
        allow_direct_play: false,
      }),
    ).toEqual(["PIN", "IP lock"]);
  });
});

describe("formatUsageStat", () => {
  it("renders unlimited caps with the infinity symbol", () => {
    expect(formatUsageStat("Plays", 2, 0)).toBe("Plays 2/∞");
  });
});

describe("buildGuestFacts", () => {
  it("returns guest-facing entitlement facts without raw target identifiers", () => {
    expect(
      buildGuestFacts({
        max_resolution: "4k",
        max_devices: 2,
        max_watch_minutes: 180,
        max_plays: 1,
      }),
    ).toEqual([
      ["Resolution", "4k"],
      ["Devices", "2"],
      ["Watch time", "180 min"],
      ["Play limit", "1"],
    ]);
  });
});
