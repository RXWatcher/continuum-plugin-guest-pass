import { describe, expect, it } from "vitest";
import {
  buildGuestFacts,
  buildPassRowMeta,
  buildPassRowTags,
  formatUsageStat,
  formatPassTargetContext,
  guestPassStatusMessage,
} from "./presentation";

describe("buildPassRowMeta", () => {
  it("includes compact target context alongside expiry and resolution", () => {
    const meta = buildPassRowMeta({
      target_type: "media_file",
      target_id: "42",
      effective_expires_at: "2026-05-20T12:00:00Z",
      max_resolution: "1080p",
    });

    expect(meta).toHaveLength(3);
    expect(meta[0]).toBe("File #42");
    expect(meta[1]).toContain("Expires ");
    expect(meta[2]).toBe("1080p");
  });
});

describe("formatPassTargetContext", () => {
  it("humanizes media file targets", () => {
    expect(formatPassTargetContext("media_file", "84")).toBe("File #84");
  });

  it("falls back to title-cased target labels for unknown target types", () => {
    expect(formatPassTargetContext("screening_room", "west-wing")).toBe("Screening room #west-wing");
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

  it("includes download and direct-play affordances when enabled", () => {
    expect(
      buildPassRowTags({
        require_pin: false,
        lock_to_first_ip: false,
        allow_downloads: true,
        allow_direct_play: true,
      }),
    ).toEqual(["Downloads", "Direct play"]);
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

  it("formats unlimited guest facts cleanly", () => {
    expect(
      buildGuestFacts({
        max_resolution: "1080p",
        max_devices: 0,
        max_watch_minutes: 0,
        max_plays: 0,
      }),
    ).toEqual([
      ["Resolution", "1080p"],
      ["Devices", "∞"],
      ["Watch time", "∞ min"],
      ["Play limit", "∞"],
    ]);
  });
});

describe("guestPassStatusMessage", () => {
  it("maps public-route statuses to human-readable guest copy", () => {
    expect(guestPassStatusMessage("expired")).toBe("This guest pass has expired.");
    expect(guestPassStatusMessage("revoked")).toBe("This guest pass has been revoked.");
    expect(guestPassStatusMessage("device_limit_reached")).toBe("This guest pass has reached its device limit.");
    expect(guestPassStatusMessage("open_limit_reached")).toBe("This guest pass has reached its open limit.");
    expect(guestPassStatusMessage("play_limit_reached")).toBe("This guest pass has reached its play limit.");
    expect(guestPassStatusMessage("concurrent_stream_limit_reached")).toBe("Too many viewers are using this guest pass right now. Try again shortly.");
    expect(guestPassStatusMessage("ip_locked")).toBe("This guest pass is locked to the network where it was first opened.");
    expect(guestPassStatusMessage("ip_not_allowed")).toBe("This guest pass is not available from your network.");
    expect(guestPassStatusMessage("country_not_allowed")).toBe("This guest pass is not available in your region.");
    expect(guestPassStatusMessage("pin_required")).toBe("Enter the access PIN to continue.");
    expect(guestPassStatusMessage("not_found")).toBe("This guest pass could not be found.");
  });

  it("returns null for unknown statuses so callers can keep their fallback copy", () => {
    expect(guestPassStatusMessage("unexpected_status")).toBeNull();
  });
});
