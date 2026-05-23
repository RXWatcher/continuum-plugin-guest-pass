import { absoluteURL } from "@/lib/api";

export async function copyToClipboard(value: string): Promise<void> {
  await navigator.clipboard.writeText(value);
}

// printInvite opens a small printable window with the share URL. Used by
// admins to physically hand a pass to a recipient (events, viewings).
// Builds the DOM with createElement + textContent so values can't escape
// their context — no document.write, no HTML interpolation.
export function printInvite(passTitle: string, shareURL: string): void {
  const invite = window.open("", "_blank", "width=760,height=640");
  if (!invite) return;
  const doc = invite.document;

  const style = doc.createElement("style");
  style.textContent = `
    body { font-family: system-ui, sans-serif; margin: 48px; color: #111; }
    h1 { margin: 0 0 12px; font-size: 32px; }
    p { font-size: 16px; line-height: 1.5; }
    code { display: block; margin-top: 24px; padding: 16px; border: 1px solid #ccc; overflow-wrap: anywhere; }
  `;
  doc.head.appendChild(style);
  doc.title = "Guest Pass Invite";

  const h1 = doc.createElement("h1");
  h1.textContent = passTitle;
  const p = doc.createElement("p");
  p.textContent = "Your Silo guest pass is ready.";
  const code = doc.createElement("code");
  code.textContent = absoluteURL(shareURL);

  doc.body.appendChild(h1);
  doc.body.appendChild(p);
  doc.body.appendChild(code);

  invite.focus();
  invite.print();
}

export function visibleWatermark(mode: string): boolean {
  return mode.split(/[,+ ]/).some((part) => ["visible", "all"].includes(part.replace("-", "_")));
}
