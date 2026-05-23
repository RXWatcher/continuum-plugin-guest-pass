import { useEffect, useState } from "react";
import { ArrowLeft, RefreshCcw } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { CreatePassForm } from "@/components/admin/CreatePassForm";
import { ConfigPanel } from "@/components/admin/ConfigPanel";
import { MediaPicker } from "@/components/admin/MediaPicker";
import { PassList } from "@/components/admin/PassList";
import { ShareResult } from "@/components/admin/ShareResult";

import { createPass, listPasses, listPassEvents, revokePass } from "@/api/passes";
import { getConfig, updateConfig } from "@/api/config";
import { absoluteURL } from "@/lib/api";
import { duplicatePassForm, type PassDraft } from "@/lib/passTools";
import { copyToClipboard } from "@/lib/share";
import type { AppConfig, CreateResponse, MediaItem, Pass, PassEvent } from "@/lib/types";

const emptyForm: PassDraft = {
  title: "",
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
};

export function Admin() {
  const [passes, setPasses] = useState<Pass[]>([]);
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<AppConfig>({ public_base_url: "", audit_retention_days: 180 });
  const [created, setCreated] = useState<CreateResponse | null>(null);
  const [selectedMedia, setSelectedMedia] = useState<MediaItem | null>(null);
  const [form, setForm] = useState<PassDraft>(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const [eventsByPass, setEventsByPass] = useState<Record<number, PassEvent[]>>({});
  const [expandedPassID, setExpandedPassID] = useState<number | null>(null);
  const [eventsLoadingID, setEventsLoadingID] = useState<number | null>(null);

  async function reload() {
    setLoading(true);
    try {
      const [list, cfg] = await Promise.all([listPasses(), getConfig()]);
      setPasses(list);
      setConfig(cfg);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load passes");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void reload();
  }, []);

  async function submitCreate() {
    if (!selectedMedia) {
      toast.error("Select a media item before creating a guest pass.");
      return;
    }
    if (selectedMedia.media_file_id <= 0) {
      toast.error("Selected media does not have a playable file.");
      return;
    }
    setSubmitting(true);
    try {
      const data = await createPass({
        ...form,
        title: form.title.trim() || selectedMedia.title,
        target_type: "media_file",
        target_id: String(selectedMedia.media_file_id),
      });
      setCreated(data);
      toast.success("Guest pass created.");
      await reload();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create pass");
    } finally {
      setSubmitting(false);
    }
  }

  async function submitConfig() {
    try {
      const cfg = await updateConfig(config);
      setConfig(cfg);
      toast.success("Settings saved.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save settings");
    }
  }

  async function handleRevoke(id: number) {
    try {
      await revokePass(id);
      toast.success("Pass revoked.");
      await reload();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to revoke pass");
    }
  }

  async function handleCopyShare(pass: Pass) {
    if (!pass.share_url) return;
    await copyToClipboard(absoluteURL(pass.share_url));
    toast.success("Share link copied.");
  }

  function handleDuplicate(pass: Pass) {
    const draft = duplicatePassForm({
      ...pass,
      expires_in_hours: 24,
      pin: "",
    });
    setForm({ ...form, ...draft });
    const fileID = Number(pass.target_id);
    if (fileID > 0) {
      setSelectedMedia({
        content_id: pass.target_id,
        media_file_id: fileID,
        type: "media_file",
        title: pass.title,
        playable: true,
      });
    }
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  async function toggleEvents(passID: number) {
    if (expandedPassID === passID) {
      setExpandedPassID(null);
      return;
    }
    setExpandedPassID(passID);
    if (eventsByPass[passID]) return;
    setEventsLoadingID(passID);
    try {
      const events = await listPassEvents(passID);
      setEventsByPass((current) => ({ ...current, [passID]: events }));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to load activity");
    } finally {
      setEventsLoadingID(null);
    }
  }

  return (
    <main className="min-h-[100dvh] bg-background text-foreground">
      <div className="mx-auto max-w-[1400px] space-y-6 px-4 py-6 md:px-6 lg:px-8">
        <header className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <a
              href="/admin/plugins"
              className="text-muted-foreground hover:bg-surface-hover hover:text-foreground inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs font-medium transition-colors"
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              Silo
            </a>
            <p className="text-xs uppercase tracking-wide text-muted-foreground">
              Plugin administration
            </p>
            <h1 className="text-2xl font-semibold">Guest Passes</h1>
            <p className="text-sm text-muted-foreground">
              Issue temporary media access links and monitor pass usage.
            </p>
          </div>
          <Button size="icon" variant="ghost" onClick={() => void reload()} aria-label="Refresh">
            <RefreshCcw className="h-4 w-4" />
          </Button>
        </header>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[3fr_2fr]">
          <section className="space-y-4">
            {created && (
              <ShareResult
                created={created}
                onCopy={() => toast.success("Share link copied.")}
              />
            )}
            <div className="rounded-lg border border-border bg-card p-5 shadow-sm">
              <div className="mb-5">
                <h2 className="text-lg font-semibold">Configuration</h2>
                <p className="text-sm text-muted-foreground">
                  Define the target, expiration, playback limits, and access controls.
                </p>
              </div>
              <div className="mb-5">
                <MediaPicker selected={selectedMedia} onSelect={setSelectedMedia} />
              </div>
              <CreatePassForm
                value={form}
                onChange={setForm}
                onSubmit={submitCreate}
                selectedTitle={selectedMedia?.title}
                submitting={submitting}
              />
            </div>
          </section>
          <aside className="space-y-4">
            <ConfigPanel value={config} onChange={setConfig} onSubmit={submitConfig} />
            <PassList
              passes={passes}
              loading={loading}
              eventsByPass={eventsByPass}
              expandedPassID={expandedPassID}
              eventsLoadingID={eventsLoadingID}
              onCopyShare={handleCopyShare}
              onDuplicate={handleDuplicate}
              onToggleEvents={toggleEvents}
              onRevoke={handleRevoke}
            />
          </aside>
        </div>
      </div>
    </main>
  );
}
