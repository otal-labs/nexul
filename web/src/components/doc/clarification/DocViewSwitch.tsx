import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { ActiveIndicator } from "@/components/ActiveIndicator";
import type { DocView } from "@/hooks/useDocView";

interface DocViewSwitchProps {
  view: DocView;
  waiting: number;
  onChange: (view: DocView) => void;
}

// The sliding highlight paints the selected side, so the items drop their own fill.
const itemClass = "data-[state=on]:bg-transparent";

// The doc page's "Doc | Questions N" switch; N is the questions waiting on an answer.
// The highlight sits beside the group, not in it, so the items keep their first and last child corners.
export const DocViewSwitch = ({ view, waiting, onChange }: DocViewSwitchProps) => (
  <div className="relative isolate w-fit">
    <ActiveIndicator selector='[data-state="on"]' className="rounded-md bg-accent" />
    <ToggleGroup
      type="single"
      variant="outline"
      size="sm"
      value={view}
      // Radix reports "" when the active item is clicked again; keep the current view.
      onValueChange={(value) => value && onChange(value as DocView)}
      aria-label="Doc or questions"
    >
      <ToggleGroupItem value="doc" className={itemClass}>
        Doc
      </ToggleGroupItem>
      <ToggleGroupItem value="questions" className={`gap-1.5 ${itemClass}`} aria-label={waiting > 0 ? `Questions, ${waiting} waiting` : "Questions"}>
        Questions
        {waiting > 0 && <span className="font-mono text-xs text-muted-foreground tabular-nums">{waiting}</span>}
      </ToggleGroupItem>
    </ToggleGroup>
  </div>
);
