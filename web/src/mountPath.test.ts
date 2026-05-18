import { describe, expect, it } from "vitest";
import { extractMountPath } from "./mountPath";

describe("extractMountPath", () => {
  it("returns an empty mount path on the dev server", () => {
    expect(extractMountPath("/admin")).toBe("");
    expect(extractMountPath("/p/token")).toBe("");
  });

  it("supports numeric and slug plugin installation ids", () => {
    expect(extractMountPath("/api/v1/plugins/42/admin")).toBe("/api/v1/plugins/42");
    expect(extractMountPath("/api/v1/plugins/guest-pass/p/token")).toBe("/api/v1/plugins/guest-pass");
  });
});
