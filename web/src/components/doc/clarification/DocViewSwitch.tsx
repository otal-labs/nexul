import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { DocView } from "@/hooks/useDocView";

interface DocViewSwitchProps {
  view: DocView;
  waiting: number;
  onChange: (view: DocView) => void;
}

// The doc page's "Doc | Questions N" switch; N is the questions waiting on an answer.
export const DocViewSwitch = ({ view, waiting, onChange }: DocViewSwitchProps) => (
  <ToggleGroup
    type="single"
    variant="segmented"
    size="xs"
    value={view}
    // Radix reports "" when the active item is clicked again; keep the current view.
    onValueChange={(value) => value && onChange(value as DocView)}
    aria-label="Doc or questions"
  >
    <ToggleGroupItem value="doc">Doc</ToggleGroupItem>
    <ToggleGroupItem value="questions" className="gap-1.5" aria-label={waiting > 0 ? `Questions, ${waiting} waiting` : "Questions"}>
      Questions
      {waiting > 0 && <span className="font-mono text-xs text-muted-foreground tabular-nums">{waiting}</span>}
    </ToggleGroupItem>
  </ToggleGroup>
);
