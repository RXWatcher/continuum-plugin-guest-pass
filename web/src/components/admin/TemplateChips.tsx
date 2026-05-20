import { passTemplates, type PassDraft } from "@/lib/passTools";

type Props = {
  onApply: (values: Partial<PassDraft>) => void;
};

// Quick-pick templates for the common pass shapes (preview, screening,
// press, family). Clicking applies the template's defaults to the form.
export function TemplateChips({ onApply }: Props) {
  return (
    <div className="grid grid-cols-2 gap-2" aria-label="Guest pass templates">
      {passTemplates.map((template) => (
        <button
          key={template.id}
          type="button"
          onClick={() => onApply(template.values)}
          className="rounded-md border border-border bg-card px-3 py-2 text-left transition-colors hover:bg-surface-hover"
        >
          <span className="block text-sm font-medium">{template.label}</span>
          <span className="mt-0.5 block text-xs text-muted-foreground">{template.description}</span>
        </button>
      ))}
    </div>
  );
}
