import { Button } from "@/components/ui/button";

interface StaleComputerNoteProps {
  days: number;
  onRemove: () => void;
}

// A computer unseen for a week stays in the order and is skipped while offline; the button lets it go, after the confirm.
export const StaleComputerNote = ({ days, onRemove }: StaleComputerNoteProps) => (
  <div role="note" className="space-y-2">
    <p className="text-xs text-warning">Not seen for {days} days. It stays in your order and is skipped while offline.</p>
    <Button type="button" variant="outline" size="sm" className="border-destructive/40 text-destructive hover:bg-destructive/10 hover:text-destructive" title="Removing it takes it out of Nexul" onClick={onRemove}>
      Remove computer
    </Button>
  </div>
);
