import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Props = {
  pin: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
};

// PinForm gates entry to a PIN-protected pass. Single field, single
// submit — keeps the recipient experience friction-light.
export function PinForm({ pin, onChange, onSubmit }: Props) {
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
      className="space-y-3 rounded-md border border-border bg-card p-4"
    >
      <div className="space-y-1.5">
        <Label htmlFor="guest-pin">Access PIN</Label>
        <Input
          id="guest-pin"
          autoFocus
          inputMode="numeric"
          value={pin}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
      <Button type="submit" className="w-full">
        Unlock pass
      </Button>
    </form>
  );
}
