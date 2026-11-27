import { Button } from "@/components/ui/button";

interface StaleComputerNoteProps {
  name: string;
  days: number;
  onRemove: () => void;
}

// A computer unseen for a week stays in the order and is skipped while offline; this is where it is let go, after a confirm.
export const StaleComputerNote = ({ name, days, onRemove }: StaleComputerNoteProps) => (
  <div role="note" className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-warning/40 px-3 py-2 text-xs">
    <span className="text-warning">Not seen for {days} days. It stays in your order and is skipped while offline. Removing it takes it out of Nexul.</span>
    <Button type="button" variant="ghost" size="sm" className="h-6 px-2 text-xs underline" onClick={onRemove}>
      Remove {name}…
    </Button>
  </div>
);
