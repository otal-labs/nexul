import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";

import { Button } from "@/components/ui/button";

interface HistoryStepperProps {
  /** The changes the body shows, the title's included; each one bar. */
  changes: number;
  step: number;
  onHop: (to: number) => void;
  /** Wide strips spell out "changes". */
  wide: boolean;
}

// Chevrons and "1 of 2 changes"; a version with nothing changed says so instead and keeps no chevrons. The first chevron's
// glyph sits on the strip's left edge, which is the body text's.
export const HistoryStepper = ({ changes, step, onHop, wide }: HistoryStepperProps) => (
  <span className="flex shrink-0 items-center gap-1.5">
    {changes > 0 && (
      <span className="-mt-0.5 -ml-2 flex items-center gap-0.5">
        <Button variant="ghost" size="icon" className="size-6" aria-label="Previous change" title="Previous change ([)" disabled={step <= 1} onClick={() => onHop(step - 1)}>
          <ChevronUpIcon className="size-4" />
        </Button>
        <Button variant="ghost" size="icon" className="size-6" aria-label="Next change" title="Next change (])" disabled={step >= changes} onClick={() => onHop(step + 1)}>
          <ChevronDownIcon className="size-4" />
        </Button>
      </span>
    )}
    <span className="text-[13px] font-medium whitespace-nowrap tabular-nums" aria-live="polite">
      {changes > 0 && `${step} of ${changes}${wide ? (changes === 1 ? " change" : " changes") : ""}`}
      {changes === 0 && "No text changes"}
    </span>
  </span>
);
