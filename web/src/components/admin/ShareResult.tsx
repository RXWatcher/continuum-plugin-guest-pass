import { Copy, Printer } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { absoluteURL } from "@/lib/api";
import { copyToClipboard, printInvite } from "@/lib/share";
import type { CreateResponse } from "@/lib/types";

type Props = {
  created: CreateResponse;
  onCopy?: () => void;
};

// ShareResult is the success card shown after creating a new pass.
// Renders the absolute share URL and exposes copy + print actions.
export function ShareResult({ created, onCopy }: Props) {
  const shareURL = absoluteURL(created.share_url);
  return (
    <Card className="border-success/30 bg-success/5">
      <CardContent className="space-y-3 py-5">
        <p className="text-xs font-medium uppercase tracking-wide text-success">
          Share link ready
        </p>
        <h3 className="text-lg font-semibold">{created.pass.title}</h3>
        <code className="block break-all rounded-md border border-border bg-background px-3 py-2 text-sm">
          {shareURL}
        </code>
        <div className="flex gap-2 pt-1">
          <Button
            size="sm"
            variant="secondary"
            onClick={async () => {
              await copyToClipboard(shareURL);
              onCopy?.();
            }}
          >
            <Copy className="mr-2 h-4 w-4" /> Copy link
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => printInvite(created.pass.title, created.share_url)}
          >
            <Printer className="mr-2 h-4 w-4" /> Print invite
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
