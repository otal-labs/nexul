import { microheaderClass } from "@/components/Microheader";
import { cn } from "@/lib/utils";

interface StepSegmentsProps {
  current: number;
  total: number;
}

// Done and current steps fill in brand, the rest stay hairlines.
export const StepSegments = ({ current, total }: StepSegmentsProps) => (
  <div role="status" aria-label={`Step ${current} of ${total}`} className="flex items-center gap-2.5">
    <span aria-hidden className={microheaderClass}>Step {current} of {total}</span>
    <span aria-hidden className="flex gap-1">
      {Array.from({ length: total }, (_, i) => (
        <span key={i} className={cn("h-0.5 w-6 rounded-full", i < current ? "bg-brand" : "bg-border")} />
      ))}
    </span>
  </div>
);
