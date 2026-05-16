import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import { Copy, ExternalLink, RefreshCcw, Shield, Trash2 } from "lucide-react";
import "./styles.css";

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

type PlayResponse = {
  message?: string;
  pass: Pass;
  stream_url?: string;
  play_method?: string;
  expires_at?: string;
  watermark?: string;
  logo_url?: string;
};

function mountPath(): string {
  const match = window.location.pathname.match(/^(\/api\/v1\/plugins\/\d+)/);
  return match ? match[1] : "";
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${mountPath()}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
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
  const [form, setForm] = useState({
    title: "",
    target_type: "movie",
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

  async function load() {
    setLoading(true);
    setError("");
    try {
      const data = await api<{ passes: Pass[] }>("/api/admin/passes");
      setPasses(data.passes ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load passes");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function createPass(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    try {
      const data = await api<CreateResponse>("/api/admin/passes", {
        method: "POST",
        body: JSON.stringify(form),
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

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Continuum</p>
          <h1>Guest Passes</h1>
        </div>
        <button className="icon-button" onClick={load} type="button" aria-label="Refresh">
          <RefreshCcw size={18} />
        </button>
      </header>

      {error && <div className="alert">{error}</div>}
      {created && (
        <div className="success">
          <strong>Pass created.</strong>
          <code>{absoluteURL(created.share_url)}</code>
          <button type="button" onClick={() => void copy(absoluteURL(created.share_url))}>
            <Copy size={16} /> Copy link
          </button>
        </div>
      )}

      <section className="grid">
        <form className="panel" onSubmit={createPass}>
          <h2>Create Pass</h2>
          <label>
            Title
            <input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} required />
          </label>
          <div className="split">
            <label>
              Target type
              <select
                value={form.target_type}
                onChange={(e) => setForm({ ...form, target_type: e.target.value })}
              >
                <option value="movie">Movie</option>
                <option value="media_file">Media file</option>
                <option value="episode">Episode</option>
                <option value="series">Series</option>
                <option value="collection">Collection</option>
                <option value="watch_room">Watch room</option>
                <option value="ebook">Ebook</option>
                <option value="audiobook">Audiobook</option>
              </select>
            </label>
            <label>
              Target ID
              <input value={form.target_id} onChange={(e) => setForm({ ...form, target_id: e.target.value })} required />
            </label>
          </div>
          <label>
            Note
            <textarea value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} rows={3} />
          </label>
          <div className="split">
            <NumberField label="Expires in hours" value={form.expires_in_hours} onChange={(v) => setForm({ ...form, expires_in_hours: v })} min={1} />
            <NumberField label="Hours after first open" value={form.valid_hours_after_first_open} onChange={(v) => setForm({ ...form, valid_hours_after_first_open: v })} min={0} />
          </div>
          <div className="split">
            <NumberField label="Max opens" value={form.max_opens} onChange={(v) => setForm({ ...form, max_opens: v })} min={0} />
            <NumberField label="Max plays" value={form.max_plays} onChange={(v) => setForm({ ...form, max_plays: v })} min={0} />
          </div>
          <div className="split">
            <NumberField label="Max watch minutes" value={form.max_watch_minutes} onChange={(v) => setForm({ ...form, max_watch_minutes: v })} min={0} />
            <NumberField label="Concurrent streams" value={form.max_concurrent_streams} onChange={(v) => setForm({ ...form, max_concurrent_streams: v })} min={1} />
          </div>
          <div className="split">
            <NumberField label="Max devices" value={form.max_devices} onChange={(v) => setForm({ ...form, max_devices: v })} min={1} />
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
          <div className="split">
            <NumberField label="Session grace minutes" value={form.session_grace_minutes} onChange={(v) => setForm({ ...form, session_grace_minutes: v })} min={0} />
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
          </div>
          <label>
            Watermark text
            <input value={form.watermark_profile} onChange={(e) => setForm({ ...form, watermark_profile: e.target.value })} placeholder="Guest pass {{pass_id}} · {{ip}} · {{time}}" />
          </label>
          <label>
            Watermark logo URL or local path
            <input value={form.watermark_logo_url} onChange={(e) => setForm({ ...form, watermark_logo_url: e.target.value })} placeholder="https://example.com/logo.png or /opt/continuum/logo.png" />
          </label>
          <label>
            IP allowlist
            <textarea value={form.ip_allowlist} onChange={(e) => setForm({ ...form, ip_allowlist: e.target.value })} rows={2} placeholder="One IP/CIDR per line, or comma separated" />
          </label>
          <div className="split">
            <label>
              Country allowlist
              <input value={form.country_allowlist} onChange={(e) => setForm({ ...form, country_allowlist: e.target.value })} placeholder="US, NL" />
            </label>
            <label>
              Geofence
              <input value={form.geofence} onChange={(e) => setForm({ ...form, geofence: e.target.value })} placeholder="US, NL" />
            </label>
          </div>
          <button className="primary" type="submit">
            <Shield size={17} /> Create guest pass
          </button>
        </form>

        <section className="panel pass-list">
          <h2>Recent Passes</h2>
          {loading ? <p className="muted">Loading...</p> : null}
          {!loading && passes.length === 0 ? <p className="muted">No passes yet.</p> : null}
          {passes.map((pass) => (
            <article className="pass-row" key={pass.id}>
              <div>
                <div className="row-title">{pass.title}</div>
                <div className="muted">{pass.target_type}:{pass.target_id}</div>
                <div className="metrics">
                  <span>{pass.status}</span>
                  <span>{pass.open_count}/{limit(pass.max_opens)} opens</span>
                  <span>{pass.play_count}/{limit(pass.max_plays)} plays</span>
                  <span>{pass.max_resolution}</span>
                  {pass.require_pin && <span>PIN</span>}
                  {pass.lock_to_first_ip && <span>IP lock</span>}
                </div>
              </div>
              <div className="row-actions">
                {pass.share_url && (
                  <button type="button" onClick={() => void copy(absoluteURL(pass.share_url!))} aria-label="Copy link">
                    <Copy size={16} />
                  </button>
                )}
                <button type="button" onClick={() => revoke(pass.id)} aria-label="Revoke">
                  <Trash2 size={16} />
                </button>
              </div>
            </article>
          ))}
        </section>
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
              <dt>Opens</dt><dd>{pass.open_count}/{limit(pass.max_opens)}</dd>
              <dt>Plays</dt><dd>{pass.play_count}/{limit(pass.max_plays)}</dd>
              <dt>Resolution</dt><dd>{pass.max_resolution}</dd>
              <dt>Devices</dt><dd>{limit(pass.max_devices)}</dd>
              <dt>Watch time</dt><dd>{limit(pass.max_watch_minutes)} min</dd>
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

function Checkbox({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <label className="check">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      {label}
    </label>
  );
}

function limit(value: number) {
  return value > 0 ? String(value) : "∞";
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
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

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
