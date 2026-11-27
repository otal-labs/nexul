import { Button } from "@/components/ui/button";

interface StaleComputerNoteProps {
  // Its place in your order, 0 when it is not in it.
  rank: number;
  onRemove: () => void;
}

// A computer unseen for a week keeps its place and is skipped while offline; the button lets it go, after the confirm.
export const StaleComputerNote = ({ rank, onRemove }: StaleComputerNoteProps) => (
  <div role="note" className="flex items-center justify-between gap-3">
    <p className="min-w-0 text-xs text-warning">Skipped while offline.{rank > 0 && ` Stays at #${rank} in your order.`}</p>
    <Button type="button" variant="outline" size="sm" className="shrink-0 border-destructive/40 text-destructive hover:bg-destructive/10 hover:text-destructive" title="Removing it takes it out of Nexul" onClick={onRemove}>
      Remove computer
    </Button>
  </div>
);
