import { useState } from "react";
import { Shield } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { PassDraft } from "@/lib/passTools";

import { TemplateChips } from "./TemplateChips";
import { PolicyPreview } from "./PolicyPreview";

type Props = {
  value: PassDraft;
  onChange: (next: PassDraft) => void;
  onSubmit: () => void;
  selectedTitle?: string;
  disabled?: boolean;
  submitting?: boolean;
};

// CreatePassForm: split into fast-path fields (top), advanced expander,
// and the live policy preview. Owns no business logic — every field
// reads from `value` and emits the new state via `onChange`.
export function CreatePassForm({ value, onChange, onSubmit, selectedTitle, disabled, submitting }: Props) {
  const [showAdvanced, setShowAdvanced] = useState(false);
  const update = (partial: Partial<PassDraft>) => onChange({ ...value, ...partial });

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
      className="space-y-6"
    >
      <TemplateChips
        onApply={(values) => {
          onChange({ ...value, ...values });
          setShowAdvanced(true);
        }}
      />

      <section className="space-y-4">
        <div className="space-y-1.5">
          <Label htmlFor="pass-title">Pass title</Label>
          <Input
            id="pass-title"
            value={value.title}
            onChange={(e) => update({ title: e.target.value })}
            placeholder={selectedTitle || ""}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="pass-note">Note</Label>
          <Textarea
            id="pass-note"
            value={value.note}
            onChange={(e) => update({ note: e.target.value })}
            rows={3}
          />
        </div>
      </section>

      <section className="grid grid-cols-3 gap-3">
        <NumberField
          label="Expires in hours"
          value={value.expires_in_hours}
          onChange={(v) => update({ expires_in_hours: v })}
          min={1}
        />
        <NumberField
          label="Max plays"
          value={value.max_plays}
          onChange={(v) => update({ max_plays: v })}
          min={0}
        />
        <div className="space-y-1.5">
          <Label htmlFor="max-resolution">Max resolution</Label>
          <Select value={value.max_resolution} onValueChange={(v) => update({ max_resolution: v })}>
            <SelectTrigger id="max-resolution">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="480p">480p</SelectItem>
              <SelectItem value="720p">720p</SelectItem>
              <SelectItem value="1080p">1080p</SelectItem>
              <SelectItem value="4k">4K</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </section>

      <PolicyPreview form={value} selectedTitle={selectedTitle} />

      <button
        type="button"
        onClick={() => setShowAdvanced((s) => !s)}
        className="text-sm text-primary underline-offset-4 hover:underline"
      >
        {showAdvanced ? "Hide advanced options" : "Advanced options"}
      </button>

      {showAdvanced && (
        <AdvancedFields value={value} onChange={onChange} />
      )}

      <div className="pt-2">
        <Button type="submit" disabled={disabled || submitting}>
          <Shield className="mr-2 h-4 w-4" />
          {submitting ? "Creating..." : "Create guest pass"}
        </Button>
      </div>
    </form>
  );
}

function AdvancedFields({ value, onChange }: { value: PassDraft; onChange: (next: PassDraft) => void }) {
  const update = (partial: Partial<PassDraft>) => onChange({ ...value, ...partial });
  return (
    <div className="space-y-6 rounded-md border border-border bg-surface p-4">
      <div className="grid grid-cols-3 gap-3">
        <NumberField
          label="Hours after first open"
          value={value.valid_hours_after_first_open}
          onChange={(v) => update({ valid_hours_after_first_open: v })}
          min={0}
        />
        <NumberField
          label="Max opens"
          value={value.max_opens}
          onChange={(v) => update({ max_opens: v })}
          min={0}
        />
        <NumberField
          label="Max watch minutes"
          value={value.max_watch_minutes}
          onChange={(v) => update({ max_watch_minutes: v })}
          min={0}
        />
        <NumberField
          label="Concurrent streams"
          value={value.max_concurrent_streams}
          onChange={(v) => update({ max_concurrent_streams: v })}
          min={1}
        />
        <NumberField
          label="Max devices"
          value={value.max_devices}
          onChange={(v) => update({ max_devices: v })}
          min={1}
        />
        <NumberField
          label="Session grace minutes"
          value={value.session_grace_minutes}
          onChange={(v) => update({ session_grace_minutes: v })}
          min={0}
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <SwitchField
          label="Require PIN"
          checked={value.require_pin}
          onChange={(v) => update({ require_pin: v })}
        />
        <SwitchField
          label="Lock to first IP"
          checked={value.lock_to_first_ip}
          onChange={(v) => update({ lock_to_first_ip: v })}
        />
        <SwitchField
          label="Allow downloads"
          checked={value.allow_downloads}
          onChange={(v) => update({ allow_downloads: v })}
        />
        <SwitchField
          label="Allow direct play"
          checked={value.allow_direct_play}
          onChange={(v) => update({ allow_direct_play: v })}
        />
        <SwitchField
          label="Disable seeking"
          checked={value.disable_seeking}
          onChange={(v) => update({ disable_seeking: v })}
        />
        <SwitchField
          label="Per-item play count"
          checked={value.per_item_play_count}
          onChange={(v) => update({ per_item_play_count: v })}
        />
      </div>

      {value.require_pin && (
        <div className="space-y-1.5">
          <Label htmlFor="pin">PIN</Label>
          <Input
            id="pin"
            value={value.pin}
            onChange={(e) => update({ pin: e.target.value })}
            required
          />
        </div>
      )}

      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1.5">
          <Label htmlFor="watermark-mode">Watermark mode</Label>
          <Select value={value.watermark_mode} onValueChange={(v) => update({ watermark_mode: v })}>
            <SelectTrigger id="watermark-mode">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">None</SelectItem>
              <SelectItem value="visible">Visible overlay</SelectItem>
              <SelectItem value="burned_in">Burned in</SelectItem>
              <SelectItem value="forensic">Forensic metadata</SelectItem>
              <SelectItem value="all">All modes</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="col-span-2 space-y-1.5">
          <Label htmlFor="watermark-template">Watermark text</Label>
          <Input
            id="watermark-template"
            value={value.watermark_profile}
            onChange={(e) => update({ watermark_profile: e.target.value })}
            placeholder="Guest pass {{pass_id}} · {{ip}} · {{time}}"
          />
        </div>
        <div className="col-span-2 space-y-1.5">
          <Label htmlFor="watermark-logo">Watermark logo URL or local path</Label>
          <Input
            id="watermark-logo"
            value={value.watermark_logo_url}
            onChange={(e) => update({ watermark_logo_url: e.target.value })}
            placeholder="https://example.com/logo.png or /opt/silo/logo.png"
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3">
        <div className="space-y-1.5">
          <Label htmlFor="ip-allowlist">IP allowlist</Label>
          <Textarea
            id="ip-allowlist"
            value={value.ip_allowlist}
            onChange={(e) => update({ ip_allowlist: e.target.value })}
            rows={2}
            placeholder="One IP/CIDR per line, or comma separated"
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label htmlFor="country-allowlist">Country allowlist</Label>
            <Input
              id="country-allowlist"
              value={value.country_allowlist}
              onChange={(e) => update({ country_allowlist: e.target.value })}
              placeholder="US, NL"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="geofence">Geofence</Label>
            <Input
              id="geofence"
              value={value.geofence}
              onChange={(e) => update({ geofence: e.target.value })}
              placeholder="US, NL"
            />
          </div>
        </div>
      </div>
    </div>
  );
}

function NumberField({
  label,
  value,
  onChange,
  min,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  min: number;
}) {
  const id = label.replace(/\s+/g, "-").toLowerCase();
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        value={value}
        min={min}
        onChange={(e) => onChange(Number(e.target.value))}
      />
    </div>
  );
}

function SwitchField({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  const id = label.replace(/\s+/g, "-").toLowerCase();
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border bg-card px-3 py-2">
      <Label htmlFor={id} className="cursor-pointer text-sm font-normal">
        {label}
      </Label>
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
    </div>
  );
}
