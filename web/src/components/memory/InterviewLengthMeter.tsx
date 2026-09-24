import { cn } from "@/lib/utils";
import { MAX_INTERVIEW_CHARS } from "@/models/Memory";

interface InterviewLengthMeterProps {
  length: number;
}

// The interview goes with every agent turn, so it has a cap; this counts against it as the author types.
export const InterviewLengthMeter = ({ length }: InterviewLengthMeterProps) => {
  const over = length > MAX_INTERVIEW_CHARS;
  return (
    <p
      role={over ? "alert" : undefined}
      className={cn("font-mono text-xs tabular-nums text-muted-foreground", over && "text-destructive")}
    >
      {length.toLocaleString("en-US")} / {MAX_INTERVIEW_CHARS.toLocaleString("en-US")} characters
      {over && " · over the cap, a save will be refused. Keep it to rules, not a transcript."}
    </p>
  );
};
