import { cn } from "@/lib/utils";

interface RunnerStatusDotProps {
  connected: boolean;
}

// Offline is a hollow ring, not destructive: going offline isn't itself an error. The row's state column says it in words.
export const RunnerStatusDot = ({ connected }: RunnerStatusDotProps) => (
  <span className="flex justify-center" title={connected ? "Online" : "Offline"}>
    <span
      aria-hidden
      className={cn(
        "size-2 rounded-full transition-colors duration-150 ease-standard",
        connected ? "bg-success" : "ring-[1.5px] ring-muted-foreground/60 ring-inset",
      )}
    />
  </span>
);
