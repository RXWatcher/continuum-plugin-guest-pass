import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import { Activity, ArrowLeft, Copy, ExternalLink, Printer, RefreshCcw, Search, Shield, Trash2 } from "lucide-react";
import { mountPath } from "./mountPath";
import { buildPolicySummary, duplicatePassForm, eventTone, passTemplates } from "./passTools";
import {
  buildGuestFacts,
  buildPassRowMeta,
  buildPassRowTags,
  formatUsageStat,
  formatShortDate,
} from "./presentation";
import "./styles.css";

let cachedToken = "";

function captureTokenFromURL(): void {
  const params = new URLSearchParams(window.location.search);
  cachedToken = params.get("token") || "";
  const theme = params.get("theme") || sessionStorage.getItem("continuum-theme") || "";
  if (theme) {
    document.documentElement.dataset.theme = theme;
    try {
      sessionStorage.setItem("continuum-theme", theme);
    } catch {
      // Ignore storage failures in private browsing contexts.
    }
  }
  if (!params.has("token")) return;
  params.delete("token");
  const clean = window.location.pathname + (params.toString() ? `?${params.toString()}` : "") + window.location.hash;
  window.history.replaceState(null, "", clean);
}

captureTokenFromURL();

function authHeaders(): Record<string, string> {
  return cachedToken ? { Authorization: `Bearer ${cachedToken}` } : {};
}

type Pass = {
  id: number;
  title: string;
  target_type: string;
  target_id: string;
  note?: string;
  created_by: string;
  expires_at: string;
  effective_expires_at: string;
  valid_hours_after_first_open: number;
  max_opens: number;
  max_plays: number;
  max_watch_minutes: number;
  max_concurrent_streams: number;
  max_devices: number;
  max_resolution: string;
  allow_downloads: boolean;
  allow_direct_play: boolean;
  lock_to_first_ip: boolean;
  require_pin: boolean;
  disable_seeking: boolean;
  watermark_mode: string;
  watermark_profile: string;
  watermark_logo_url: string;
  ip_allowlist: string[];
  country_allowlist: string[];
  session_grace_minutes: number;
  per_item_play_count: boolean;
  geofence: string[];
  open_count: number;
  play_count: number;
  first_opened_at?: string;
  revoked_at?: string;
  created_at: string;
  status: string;
  share_url?: string;
};

type CreateResponse = {
  pass: Pass;
  token: string;
  share_url: string;
};

type PassEvent = {
  id: number;
  pass_id: number;
  type: string;
  ip?: string;
  user_agent?: string;
  attrs?: Record<string, unknown>;
  created_at: string;
};

type PlayResponse = {
  message?: string;
  pass: Pass;
  stream_url?: string;
  play_method?: string;
  expires_at?: string;
  watermark?: string;
  logo_url?: string;
};

type AppConfig = {
  public_base_url: string;
  audit_retention_days: number;
};

type MediaItem = {
  content_id?: string;
  MediaID?: string;
  media_id?: string;
  media_file_id?: number;
  playable?: boolean;
  type?: string;
  MediaType?: string;
  media_type?: string;
  Title?: string;
  title?: string;
  Year?: number;
  year?: number;
  Overview?: string;
  overview?: string;
  PosterURL?: string;
  poster_url?: string;
  RuntimeMinutes?: number;
  runtime_minutes?: number;
  ContentRating?: string;
  content_rating?: string;
};

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${mountPath()}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...authHeaders(), ...(init?.headers ?? {}) },
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const message = data?.error?.message || data?.message || `Request failed (${res.status})`;
    throw new Error(message);
  }
  return data as T;
}

function App() {
  if (window.location.pathname.includes("/p/")) {
    return <GuestPassPage />;
  }
  return <AdminPage />;
}

function AdminPage() {
  const [passes, setPasses] = useState<Pass[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [created, setCreated] = useState<CreateResponse | null>(null);
  const [mediaQuery, setMediaQuery] = useState("");
  const [mediaType, setMediaType] = useState("all");
  const [mediaResults, setMediaResults] = useState<MediaItem[]>([]);
  const [selectedMedia, setSelectedMedia] = useState<MediaItem | null>(null);
  const [resolvingMediaId, setResolvingMediaId] = useState("");
  const [mediaLoading, setMediaLoading] = useState(false);
  const [mediaError, setMediaError] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [eventsByPass, setEventsByPass] = useState<Record<number, PassEvent[]>>({});
  const [activeEventsPassID, setActiveEventsPassID] = useState<number | null>(null);
  const [eventLoading, setEventLoading] = useState<number | null>(null);
  const [config, setConfig] = useState<AppConfig>({ public_base_url: "", audit_retention_days: 180 });
  const [configStatus, setConfigStatus] = useState("");
  const [form, setForm] = useState({
    title: "",
    target_type: "media_file",
    target_id: "",
    note: "",
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
    require_pin: false,
    pin: "",
    disable_seeking: false,
    watermark_mode: "none",
    watermark_profile: "",
    watermark_logo_url: "",
    ip_allowlist: "",
    country_allowlist: "",
    session_grace_minutes: 0,
    per_item_play_count: false,
    geofence: "",
  });
  const policySummary = useMemo(() => buildPolicySummary(form), [form]);

  async function load() {
    setLoading(true);
    setError("");
    try {
      const data = await api<{ passes: Pass[] }>("/api/admin/passes");
      setPasses(data.passes ?? []);
      const cfg = await api<AppConfig>("/api/admin/config");
      setConfig(cfg);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load passes");
    } finally {
      setLoading(false);
    }
  }

  async function saveConfig(event: React.FormEvent) {
    event.preventDefault();
    setConfigStatus("");
    setError("");
    try {
      const cfg = await api<AppConfig>("/api/admin/config", {
        method: "PATCH",
        body: JSON.stringify(config),
      });
      setConfig(cfg);
      setConfigStatus("Settings saved.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save settings");
    }
  }

  useEffect(() => {
    void load();
  }, []);

  useEffect(() => {
    const query = mediaQuery.trim();
    if (query.length < 2) {
      setMediaResults([]);
      setMediaLoading(false);
      setMediaError("");
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      setMediaLoading(true);
      setMediaError("");
      const params = new URLSearchParams({ q: query, limit: "20" });
      if (mediaType !== "all") {
        params.set("type", mediaType);
      }
      api<{ items?: MediaItem[] }>(`/api/admin/catalog/search?${params.toString()}`, { signal: controller.signal })
        .then((data) => setMediaResults(data.items ?? []))
        .catch((err) => {
          if (err instanceof DOMException && err.name === "AbortError") return;
          setMediaError(err instanceof Error ? err.message : "Media search failed");
        })
        .finally(() => {
          if (!controller.signal.aborted) {
            setMediaLoading(false);
          }
        });
    }, 250);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [mediaQuery, mediaType]);

  async function createPass(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (!selectedMedia) {
      setError("Select a media item before creating a guest pass.");
      return;
    }
    if (mediaFileID(selectedMedia) <= 0) {
      setError("Selected media does not have a playable file.");
      return;
    }
    try {
      const data = await api<CreateResponse>("/api/admin/passes", {
        method: "POST",
        body: JSON.stringify({
          ...form,
          title: form.title.trim() || mediaTitle(selectedMedia),
          target_type: "media_file",
          target_id: String(mediaFileID(selectedMedia)),
        }),
      });
      setCreated(data);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create pass");
    }
  }

  async function revoke(id: number) {
    setError("");
    try {
      await api(`/api/admin/passes/${id}/revoke`, { method: "POST" });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke pass");
    }
  }

  function applyTemplate(templateID: string) {
    const template = passTemplates.find((item) => item.id === templateID);
    if (!template) return;
    setForm((current) => ({ ...current, ...template.values }));
    setShowAdvanced(true);
  }

  function duplicate(pass: Pass) {
    const draft = duplicatePassForm({
      ...pass,
      expires_in_hours: 24,
      pin: "",
      ip_allowlist: pass.ip_allowlist,
      country_allowlist: pass.country_allowlist,
      geofence: pass.geofence,
    });
    setForm((current) => ({ ...current, ...draft, target_type: "media_file" }));
    const fileID = Number(pass.target_id);
    if (fileID > 0) {
      setSelectedMedia({
        title: pass.title,
        type: "media_file",
        media_file_id: fileID,
      });
    }
    setShowAdvanced(true);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  async function loadEvents(passID: number) {
    if (activeEventsPassID === passID) {
      setActiveEventsPassID(null);
      return;
    }
    setActiveEventsPassID(passID);
    if (eventsByPass[passID]) return;
    setEventLoading(passID);
    setError("");
    try {
      const data = await api<{ events: PassEvent[] }>(`/api/admin/passes/${passID}/events`);
      setEventsByPass((current) => ({ ...current, [passID]: data.events ?? [] }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load pass activity");
    } finally {
      setEventLoading(null);
    }
  }

  async function selectMedia(item: MediaItem) {
    if (mediaFileID(item) <= 0) {
      setMediaError("Selected media does not have a playable file.");
      return;
    }
    setResolvingMediaId(mediaID(item));
    setMediaError("");
    setSelectedMedia(item);
    setForm({ ...form, title: form.title || mediaTitle(item) });
    setResolvingMediaId("");
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <a className="back-link" href="/admin/plugins" title="Back to Continuum plugins">
            <ArrowLeft size={16} />
            <span>Continuum</span>
          </a>
          <p className="eyebrow">Plugin administration</p>
          <h1>Guest Passes</h1>
          <p className="subhead">Issue temporary media access links and monitor pass usage.</p>
        </div>
        <button className="icon-button" onClick={load} type="button" aria-label="Refresh">
          <RefreshCcw size={18} />
        </button>
      </header>

      {error && <div className="alert">{error}</div>}

      <section className="workspace">
        <div className="creation-rail">
          {created && <ShareResult created={created} />}
          <form className="panel creation-panel" onSubmit={createPass}>
            <div className="panel-heading">
              <div>
                <h2>Configuration</h2>
                <p className="panel-subtitle">Define the target, expiration, playback limits, and access controls.</p>
              </div>
              <span className="badge">New pass</span>
            </div>
            <div className="template-grid" aria-label="Guest pass templates">
              {passTemplates.map((template) => (
                <button key={template.id} type="button" onClick={() => applyTemplate(template.id)}>
                  <span>
                    <strong>{template.label}</strong>
                    <small>{template.description}</small>
                  </span>
                </button>
              ))}
            </div>
            <div className="form-section">
              <div className="media-picker wide-field">
                <div className="media-search-row">
                  <label>
                    Search media
                    <div className="search-input">
                      <Search size={16} />
                      <input value={mediaQuery} onChange={(e) => setMediaQuery(e.target.value)} placeholder="Title, episode, collection..." />
                    </div>
                  </label>
                  <label>
                    Type
                    <select value={mediaType} onChange={(e) => setMediaType(e.target.value)}>
                      <option value="all">All</option>
                      <option value="movie">Movies</option>
                      <option value="episode">Episodes</option>
                    </select>
                  </label>
                </div>
                {selectedMedia && (
                  <div className="selected-media">
                    <MediaThumb item={selectedMedia} />
                    <div>
                      <strong>{mediaTitle(selectedMedia)}</strong>
                      <span>{mediaMeta(selectedMedia)}</span>
                    </div>
                  </div>
                )}
                {mediaError && <p className="field-error">{mediaError}</p>}
                {mediaLoading && <p className="muted">Searching...</p>}
                {!mediaLoading && mediaQuery.trim().length >= 2 && mediaResults.length === 0 && !mediaError ? <p className="muted">No matches.</p> : null}
                {mediaResults.length > 0 && (
                  <div className="media-results">
                    {mediaResults.map((item) => (
                      <button
                        className={selectedMedia && mediaID(selectedMedia) === mediaID(item) ? "media-result selected" : "media-result"}
                        disabled={resolvingMediaId === mediaID(item)}
                        key={`${mediaTargetType(item)}:${mediaID(item)}`}
                        onClick={() => void selectMedia(item)}
                        type="button"
                      >
                        <MediaThumb item={item} />
                        <span>
                          <strong>{mediaTitle(item)}</strong>
                          <small>{resolvingMediaId === mediaID(item) ? "Selecting..." : mediaMeta(item)}</small>
                        </span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <label className="wide-field">
                Pass title
                <input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} placeholder={selectedMedia ? mediaTitle(selectedMedia) : ""} />
              </label>
              <label className="wide-field">
                Note
                <textarea value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} rows={3} />
              </label>
            </div>
            <div className="form-section">
              <NumberField label="Expires in hours" value={form.expires_in_hours} onChange={(v) => setForm({ ...form, expires_in_hours: v })} min={1} />
              <NumberField label="Max plays" value={form.max_plays} onChange={(v) => setForm({ ...form, max_plays: v })} min={0} />
              <label>
                Max resolution
                <select value={form.max_resolution} onChange={(e) => setForm({ ...form, max_resolution: e.target.value })}>
                  <option value="480p">480p</option>
                  <option value="720p">720p</option>
                  <option value="1080p">1080p</option>
                  <option value="4k">4K</option>
                </select>
              </label>
            </div>
            <div className="policy-preview">
              <div>
                <strong>Policy preview</strong>
                <span>{selectedMedia ? mediaTitle(selectedMedia) : "Select media to complete the pass."}</span>
              </div>
              <ul>
                {policySummary.map((item) => <li key={item}>{item}</li>)}
              </ul>
            </div>
            <button className="text-button" type="button" onClick={() => setShowAdvanced((value) => !value)}>
              {showAdvanced ? "Hide advanced options" : "Advanced options"}
            </button>
            {showAdvanced && (
              <>
                <div className="form-section">
                  <NumberField label="Hours after first open" value={form.valid_hours_after_first_open} onChange={(v) => setForm({ ...form, valid_hours_after_first_open: v })} min={0} />
                  <NumberField label="Max opens" value={form.max_opens} onChange={(v) => setForm({ ...form, max_opens: v })} min={0} />
                  <NumberField label="Max watch minutes" value={form.max_watch_minutes} onChange={(v) => setForm({ ...form, max_watch_minutes: v })} min={0} />
                  <NumberField label="Concurrent streams" value={form.max_concurrent_streams} onChange={(v) => setForm({ ...form, max_concurrent_streams: v })} min={1} />
                  <NumberField label="Max devices" value={form.max_devices} onChange={(v) => setForm({ ...form, max_devices: v })} min={1} />
                  <NumberField label="Session grace minutes" value={form.session_grace_minutes} onChange={(v) => setForm({ ...form, session_grace_minutes: v })} min={0} />
                </div>
                <div className="checkbox-grid">
                  <Checkbox label="Require PIN" checked={form.require_pin} onChange={(v) => setForm({ ...form, require_pin: v })} />
                  <Checkbox label="Lock to first IP" checked={form.lock_to_first_ip} onChange={(v) => setForm({ ...form, lock_to_first_ip: v })} />
                  <Checkbox label="Allow downloads" checked={form.allow_downloads} onChange={(v) => setForm({ ...form, allow_downloads: v })} />
                  <Checkbox label="Allow direct play" checked={form.allow_direct_play} onChange={(v) => setForm({ ...form, allow_direct_play: v })} />
                  <Checkbox label="Disable seeking" checked={form.disable_seeking} onChange={(v) => setForm({ ...form, disable_seeking: v })} />
                  <Checkbox label="Per-item play count" checked={form.per_item_play_count} onChange={(v) => setForm({ ...form, per_item_play_count: v })} />
                </div>
                {form.require_pin && (
                  <label>
                    PIN
                    <input value={form.pin} onChange={(e) => setForm({ ...form, pin: e.target.value })} required={form.require_pin} />
                  </label>
                )}
                <div className="form-section">
                  <label>
                    Watermark mode
                    <select value={form.watermark_mode} onChange={(e) => setForm({ ...form, watermark_mode: e.target.value })}>
                      <option value="none">None</option>
                      <option value="visible">Visible overlay</option>
                      <option value="burned_in">Burned in</option>
                      <option value="forensic">Forensic metadata</option>
                      <option value="all">All modes</option>
                    </select>
                  </label>
                  <label className="wide-field">
                    Watermark text
                    <input value={form.watermark_profile} onChange={(e) => setForm({ ...form, watermark_profile: e.target.value })} placeholder="Guest pass {{pass_id}} · {{ip}} · {{time}}" />
                  </label>
                  <label className="wide-field">
                    Watermark logo URL or local path
                    <input value={form.watermark_logo_url} onChange={(e) => setForm({ ...form, watermark_logo_url: e.target.value })} placeholder="https://example.com/logo.png or /opt/continuum/logo.png" />
                  </label>
                  <label className="wide-field">
                    IP allowlist
                    <textarea value={form.ip_allowlist} onChange={(e) => setForm({ ...form, ip_allowlist: e.target.value })} rows={2} placeholder="One IP/CIDR per line, or comma separated" />
                  </label>
                  <label>
                    Country allowlist
                    <input value={form.country_allowlist} onChange={(e) => setForm({ ...form, country_allowlist: e.target.value })} placeholder="US, NL" />
                  </label>
                  <label>
                    Geofence
                    <input value={form.geofence} onChange={(e) => setForm({ ...form, geofence: e.target.value })} placeholder="US, NL" />
                  </label>
                </div>
              </>
            )}
            <div className="form-actions">
              <button className="primary" type="submit">
                <Shield size={17} /> Create guest pass
              </button>
            </div>
          </form>
        </div>

        <aside className="monitoring-rail">
          <form className="panel utility-panel" onSubmit={saveConfig}>
            <div className="panel-heading">
              <div>
                <h2>Plugin settings</h2>
                <p className="panel-subtitle">Control share link generation and audit event retention.</p>
              </div>
            </div>
            <div className="form-section">
              <label>
                Public base URL
                <input
                  value={config.public_base_url}
                  onChange={(e) => setConfig({ ...config, public_base_url: e.target.value })}
                  placeholder="https://example.com/api/v1/plugins/guest-pass"
                />
              </label>
              <label>
                Audit retention days
                <input
                  min={1}
                  type="number"
                  value={config.audit_retention_days}
                  onChange={(e) => setConfig({ ...config, audit_retention_days: Number(e.target.value) })}
                />
              </label>
            </div>
            <div className="form-actions">
              <button type="submit">
                <Shield size={16} /> Save settings
              </button>
              {configStatus && <span className="muted">{configStatus}</span>}
            </div>
          </form>

          <section className="panel pass-list">
            <div className="panel-heading">
              <h2>Recent Passes</h2>
              <span className="badge">{passes.length} total</span>
            </div>
            <div className="metric-strip">
              <Metric label="Active" value={String(passes.filter((pass) => pass.status === "active").length)} />
              <Metric label="Revoked" value={String(passes.filter((pass) => pass.revoked_at || pass.status === "revoked").length)} />
              <Metric label="Opened" value={String(passes.reduce((sum, pass) => sum + pass.open_count, 0))} />
            </div>
            {loading ? <p className="muted">Loading...</p> : null}
            {!loading && passes.length === 0 ? <p className="muted">No passes yet.</p> : null}
            {passes.map((pass) => (
              <article className="pass-row" key={pass.id}>
                <div className="pass-row-copy">
                  <div className="row-title">{pass.title}</div>
                  <div className="row-meta">
                    {buildPassRowMeta(pass).map((item) => (
                      <span key={item}>{item}</span>
                    ))}
                  </div>
                  <div className="metrics">
                    <span>{pass.status}</span>
                    <span>{formatUsageStat("Opens", pass.open_count, pass.max_opens)}</span>
                    <span>{formatUsageStat("Plays", pass.play_count, pass.max_plays)}</span>
                    {buildPassRowTags(pass).map((item) => (
                      <span key={item}>{item}</span>
                    ))}
                  </div>
                </div>
                <div className="row-actions">
                  {pass.share_url && (
                    <button type="button" onClick={() => void copy(absoluteURL(pass.share_url!))}>
                      <Copy size={15} />
                      <span>Copy link</span>
                    </button>
                  )}
                  <button type="button" onClick={() => duplicate(pass)}>
                    <Copy size={15} />
                    <span>Duplicate</span>
                  </button>
                  <button type="button" onClick={() => void loadEvents(pass.id)}>
                    <Activity size={15} />
                    <span>Activity</span>
                  </button>
                  <button className="danger-button" type="button" onClick={() => revoke(pass.id)}>
                    <Trash2 size={15} />
                    <span>Revoke</span>
                  </button>
                </div>
                {activeEventsPassID === pass.id && (
                  <div className="event-timeline">
                    {eventLoading === pass.id ? <p className="muted">Loading activity...</p> : null}
                    {eventLoading !== pass.id && (eventsByPass[pass.id]?.length ?? 0) === 0 ? <p className="muted">No activity recorded.</p> : null}
                    {(eventsByPass[pass.id] ?? []).map((event) => (
                      <div className={`event-row ${eventTone(event.type)}`} key={event.id}>
                        <span>{event.type.replace(/_/g, " ")}</span>
                        <small>{formatDate(event.created_at)}{event.ip ? ` · ${event.ip}` : ""}</small>
                      </div>
                    ))}
                  </div>
                )}
              </article>
            ))}
          </section>
        </aside>
      </section>
    </main>
  );
}

function GuestPassPage() {
  const token = useMemo(() => window.location.pathname.split("/p/")[1]?.split("/")[0] || "", []);
  const deviceId = useMemo(() => guestDeviceID(), []);
  const [pass, setPass] = useState<Pass | null>(null);
  const [status, setStatus] = useState("loading");
  const [message, setMessage] = useState("");
  const [pin, setPin] = useState("");
  const [needsPin, setNeedsPin] = useState(false);
  const [playback, setPlayback] = useState<PlayResponse | null>(null);

  useEffect(() => {
    async function openPass() {
      try {
        const preview = await api<{ pass: Pass }>(`/api/public/passes/${token}`);
        if (preview.pass.require_pin) {
          setPass(preview.pass);
          setStatus(preview.pass.status);
          setNeedsPin(true);
          return;
        }
        const data = await api<{ pass: Pass }>(`/api/public/passes/${token}/open`, {
          method: "POST",
          body: JSON.stringify({ device_id: deviceId }),
        });
        setPass(data.pass);
        setStatus(data.pass.status);
      } catch (err) {
        setStatus("unavailable");
        setMessage(err instanceof Error ? err.message : "This guest pass is unavailable.");
      }
    }
    void openPass();
  }, [token]);

  async function play() {
    try {
      const data = await api<PlayResponse>(`/api/public/passes/${token}/play`, {
        method: "POST",
        body: JSON.stringify({ pin, device_id: deviceId }),
      });
      setPass(data.pass);
      if (data.stream_url) {
        setPlayback(data);
        return;
      }
      setMessage(data.message || "Playback is not available.");
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "Playback is not available.");
    }
  }

  return (
    <main className="guest-shell">
      <section className="guest-panel">
        <p className="eyebrow">Continuum Guest Pass</p>
        <h1>{pass?.title ?? "Guest Pass"}</h1>
        {playback?.stream_url ? (
          <div className="player-shell">
            <video controls autoPlay src={new URL(playback.stream_url, window.location.origin).toString()} />
            {visibleWatermark(playback.pass.watermark_mode) && (
              <>
                {playback.logo_url && <img className="watermark-logo" src={playback.logo_url} alt="" />}
                {playback.watermark && <div className="watermark-text">{playback.watermark}</div>}
              </>
            )}
          </div>
        ) : pass ? (
          <>
            {needsPin && (
              <form className="pin-row" onSubmit={(event) => {
                event.preventDefault();
                api<{ pass: Pass }>(`/api/public/passes/${token}/open`, {
                  method: "POST",
                  body: JSON.stringify({ pin, device_id: deviceId }),
                }).then((data) => {
                  setPass(data.pass);
                  setStatus(data.pass.status);
                  setNeedsPin(false);
                  setMessage("");
                }).catch((err) => setMessage(err instanceof Error ? err.message : "Invalid PIN"));
              }}>
                <label>
                  PIN
                  <input value={pin} onChange={(event) => setPin(event.target.value)} autoFocus />
                </label>
                <button className="primary" type="submit">Unlock</button>
              </form>
            )}
            <dl className="details">
              <dt>Status</dt><dd>{status}</dd>
              <dt>Target</dt><dd>{pass.target_type}:{pass.target_id}</dd>
              <dt>Expires</dt><dd>{formatDate(pass.effective_expires_at)}</dd>
              {buildGuestFacts(pass).map(([label, value]) => (
                <React.Fragment key={label}>
                  <dt>{label}</dt>
                  <dd>{value}</dd>
                </React.Fragment>
              ))}
            </dl>
            {pass.note && <p className="note">{pass.note}</p>}
            <button className="primary" type="button" onClick={play} disabled={pass.status !== "active" || needsPin}>
              <ExternalLink size={17} /> Start playback
            </button>
          </>
        ) : (
          <p className="muted">{message || "Opening pass..."}</p>
        )}
        {message && <div className="alert">{message}</div>}
      </section>
    </main>
  );
}

function NumberField({ label, value, onChange, min }: { label: string; value: number; onChange: (value: number) => void; min: number }) {
  return (
    <label>
      {label}
      <input type="number" value={value} min={min} onChange={(e) => onChange(Number(e.target.value))} />
    </label>
  );
}

function MediaThumb({ item }: { item: MediaItem }) {
  const poster = mediaPoster(item);
  if (poster) {
    return <img className="media-thumb" src={poster} alt="" />;
  }
  return <span className="media-thumb media-thumb-fallback">{mediaTitle(item).slice(0, 1).toUpperCase()}</span>;
}

function mediaID(item: MediaItem) {
  return item.content_id || item.media_id || item.MediaID || "";
}

function mediaFileID(item: MediaItem) {
  return item.media_file_id ?? 0;
}

function mediaTargetType(item: MediaItem) {
  const raw = (item.type || item.media_type || item.MediaType || "movie").toLowerCase();
  if (raw === "tv") return "series";
  return raw;
}

function mediaTitle(item: MediaItem) {
  return item.title || item.Title || "Untitled media";
}

function mediaPoster(item: MediaItem) {
  return item.poster_url || item.PosterURL || "";
}

function mediaMeta(item: MediaItem) {
  const parts = [mediaTargetType(item)];
  const year = item.year ?? item.Year;
  const runtime = item.runtime_minutes ?? item.RuntimeMinutes;
  const rating = item.content_rating || item.ContentRating;
  if (year) parts.push(String(year));
  if (runtime) parts.push(`${runtime} min`);
  if (rating) parts.push(rating);
  const id = mediaID(item);
  if (id) parts.push(id);
  const fileID = mediaFileID(item);
  if (fileID) parts.push(`file ${fileID}`);
  return parts.join(" · ");
}

function Checkbox({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <label className="check">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      {label}
    </label>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function ShareResult({ created }: { created: CreateResponse }) {
  const shareURL = absoluteURL(created.share_url);

  return (
    <section className="success share-result">
      <div>
        <p className="eyebrow">Share link ready</p>
        <h2>{created.pass.title}</h2>
        <code>{shareURL}</code>
      </div>
      <div className="share-actions">
        <button type="button" onClick={() => void copy(shareURL)}>
          <Copy size={16} />
          <span>Copy link</span>
        </button>
        <button type="button" onClick={() => printInvite(created)}>
          <Printer size={16} />
          <span>Print invite</span>
        </button>
      </div>
    </section>
  );
}

function formatDate(value: string) {
  return formatShortDate(value);
}

function absoluteURL(url: string) {
  if (url.startsWith("/") && mountPath()) {
    return new URL(`${mountPath()}${url}`, window.location.origin).toString();
  }
  return new URL(url, window.location.href).toString();
}

function visibleWatermark(mode: string) {
  return mode.split(/[,+ ]/).some((part) => ["visible", "all"].includes(part.replace("-", "_")));
}

function guestDeviceID() {
  const key = "continuum.guestPass.deviceId";
  try {
    const existing = window.localStorage.getItem(key);
    if (existing) {
      return existing;
    }
    const id = crypto.randomUUID();
    window.localStorage.setItem(key, id);
    return id;
  } catch {
    return crypto.randomUUID();
  }
}

async function copy(value: string) {
  await navigator.clipboard.writeText(value);
}

function printInvite(created: CreateResponse) {
  const invite = window.open("", "_blank", "width=760,height=640");
  if (!invite) return;
  const url = absoluteURL(created.share_url);
  invite.document.write(`<!doctype html>
<html>
<head>
  <title>Guest Pass Invite</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 48px; color: #111; }
    h1 { margin: 0 0 12px; font-size: 32px; }
    p { font-size: 16px; line-height: 1.5; }
    code { display: block; margin-top: 24px; padding: 16px; border: 1px solid #ccc; overflow-wrap: anywhere; }
  </style>
</head>
<body>
  <h1>${escapeHTML(created.pass.title)}</h1>
  <p>Your Continuum guest pass is ready.</p>
  <code>${escapeHTML(url)}</code>
</body>
</html>`);
  invite.document.close();
  invite.focus();
  invite.print();
}

function escapeHTML(value: string) {
  return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#039;" })[char] || char);
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
