import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { DocView } from "@/hooks/useDocView";

interface DocViewSwitchProps {
  view: DocView;
  ticketCount: number;
  questions: boolean;
  waiting: number;
  onChange: (view: DocView) => void;
}

const countClass = "font-mono text-xs text-muted-foreground tabular-nums";

// The doc page's "Doc | Tickets N | Questions N" switch; Tickets counts the doc's tickets, Questions those waiting on an answer.
export const DocViewSwitch = ({ view, ticketCount, questions, waiting, onChange }: DocViewSwitchProps) => (
  <ToggleGroup
    type="single"
    variant="segmented"
    size="xs"
    value={view}
    // Radix reports "" when the active item is clicked again; keep the current view.
    onValueChange={(value) => value && onChange(value as DocView)}
    aria-label="Doc views"
  >
    <ToggleGroupItem value="doc" className="h-6">Doc</ToggleGroupItem>
    {ticketCount > 0 && (
      <ToggleGroupItem value="tickets" className="h-6 gap-1.5" aria-label={`Tickets, ${ticketCount}`}>
        Tickets
        <span className={countClass}>{ticketCount}</span>
      </ToggleGroupItem>
    )}
    {questions && (
      <ToggleGroupItem value="questions" className="h-6 gap-1.5" aria-label={waiting > 0 ? `Questions, ${waiting} waiting` : "Questions"}>
        Questions
        {waiting > 0 && <span className={countClass}>{waiting}</span>}
      </ToggleGroupItem>
    )}
  </ToggleGroup>
);
