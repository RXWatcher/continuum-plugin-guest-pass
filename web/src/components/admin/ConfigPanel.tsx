import { Shield } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { AppConfig } from "@/lib/types";

type Props = {
  value: AppConfig;
  onChange: (next: AppConfig) => void;
  onSubmit: () => void;
};

// ConfigPanel edits the singleton plugin config (public base URL, audit
// retention). Wired to the typed PATCH on submit.
export function ConfigPanel({ value, onChange, onSubmit }: Props) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Plugin settings</CardTitle>
        <p className="text-sm text-muted-foreground">
          Share link generation and audit retention.
        </p>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
          className="space-y-4"
        >
          <div className="space-y-1.5">
            <Label htmlFor="public-base-url">Public base URL</Label>
            <Input
              id="public-base-url"
              value={value.public_base_url}
              onChange={(e) => onChange({ ...value, public_base_url: e.target.value })}
              placeholder="https://example.com/api/v1/plugins/guest-pass"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="audit-retention">Audit retention days</Label>
            <Input
              id="audit-retention"
              type="number"
              min={1}
              value={value.audit_retention_days}
              onChange={(e) => onChange({ ...value, audit_retention_days: Number(e.target.value) })}
            />
          </div>
          <div>
            <Button type="submit" size="sm" variant="secondary">
              <Shield className="mr-2 h-4 w-4" /> Save settings
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
