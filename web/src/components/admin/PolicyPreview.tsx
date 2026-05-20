import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { buildPolicySummary, type PassDraft } from "@/lib/passTools";

type Props = {
  form: PassDraft;
  selectedTitle?: string;
};

// PolicyPreview renders the operator-facing summary of "what this pass
// will allow / require / restrict" before they hit Create. Pure render —
// the rule logic lives in lib/passTools so it can be unit-tested.
export function PolicyPreview({ form, selectedTitle }: Props) {
  const items = buildPolicySummary(form);
  return (
    <Card className="bg-surface">
      <CardHeader>
        <CardTitle className="text-sm">Policy preview</CardTitle>
        <p className="text-xs text-muted-foreground">
          {selectedTitle || "Select media to complete the pass."}
        </p>
      </CardHeader>
      <CardContent>
        <ul className="space-y-1 text-sm text-muted-foreground">
          {items.map((item) => (
            <li key={item} className="leading-snug">
              • {item}
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
