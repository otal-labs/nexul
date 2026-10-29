import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { HistoryView } from "@/models/History";

interface HistoryViewToggleProps {
  view: HistoryView;
  onView: (view: HistoryView) => void;
}

// The picked version with its changes marked, or as it read.
export const HistoryViewToggle = ({ view, onView }: HistoryViewToggleProps) => (
  <ToggleGroup
    type="single"
    variant="segmented"
    size="xs"
    aria-label="Version view"
    value={view}
    // Radix reports "" when the chosen side is pressed again; the view stands.
    onValueChange={(next) => next && onView(next as HistoryView)}
    className="shrink-0"
  >
    <ToggleGroupItem value="changes">Changes</ToggleGroupItem>
    <ToggleGroupItem value="as-it-was">As it was</ToggleGroupItem>
  </ToggleGroup>
);
