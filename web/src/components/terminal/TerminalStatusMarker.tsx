import { cn } from "@/lib/utils";
import type { TerminalState } from "@/lib/terminal";

const describe = ({ status, exit }: TerminalState) => {
  if (status === "connecting") return { label: "Connecting…", dot: "bg-muted-foreground/50" };
  if (status === "live") return { label: "Live", dot: "bg-success" };
  if (status === "lost") return { label: "Disconnected", dot: "bg-warning-mark" };
  return {
    label: `Ended, exit ${exit}`,
    dot: exit === 0 ? "bg-muted-foreground/50" : "bg-destructive",
  };
};

// The terminal's connection in one mono word or two, as the log view's marker reads.
export const TerminalStatusMarker = ({ state }: { state: TerminalState }) => {
  const { label, dot } = describe(state);
  return (
    <span role="status" className="flex items-center gap-2 font-mono text-xs text-muted-foreground">
      <span aria-hidden className={cn("size-1.5 shrink-0 rounded-full", dot)} />
      {label}
    </span>
  );
};
