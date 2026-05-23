// Captures a one-shot bearer token from the URL on first load, strips it
// from the address bar via history.replaceState, and keeps it in memory
// for subsequent API calls. The token is *only* in JS memory — it is not
// stored in localStorage/sessionStorage, so browser history doesn't keep
// a copy after the tab closes.
//
// Silo sometimes hands operators an admin URL with ?token=... that
// the plugin needs to authenticate against; the strip keeps it from
// leaking through referrers or screen-share captures.
let cachedToken = "";

export function captureTokenFromURL(): void {
  const params = new URLSearchParams(window.location.search);
  cachedToken = params.get("token") || "";

  const theme = params.get("theme") || sessionStorage.getItem("silo-theme") || "";
  if (theme) {
    document.documentElement.dataset.theme = theme;
    try {
      sessionStorage.setItem("silo-theme", theme);
    } catch {
      // Ignore storage failures in private browsing contexts.
    }
  }

  if (!params.has("token")) return;
  params.delete("token");
  const clean =
    window.location.pathname +
    (params.toString() ? `?${params.toString()}` : "") +
    window.location.hash;
  window.history.replaceState(null, "", clean);
}

export function authHeaders(): Record<string, string> {
  return cachedToken ? { Authorization: `Bearer ${cachedToken}` } : {};
}
