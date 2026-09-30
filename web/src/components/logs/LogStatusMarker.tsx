import { useMessageScrollerScrollable } from "@/components/ui/message-scroller";
import { cn } from "@/lib/utils";
import type { LogStatus } from "@/models/ContainerLog";

interface LogStatusMarkerProps {
  status: LogStatus;
  reason: string | undefined;
  paused: boolean;
}

interface Marker {
  label: string;
  dot: string;
}

const LIVE_DOT = "bg-success animate-[status-pulse_2.4s_ease-standard_infinite]";
const IDLE_DOT = "bg-muted-foreground/50";

// Connection trouble outranks the reader's own state: Paused is the button, Scrolled up is following let go by scrolling.
const describeMarker = (status: LogStatus, paused: boolean, scrolledUp: boolean): Marker => {
  if (status === "offline") return { label: "Runner offline, reconnecting…", dot: "bg-warning" };
  if (status === "ended") return { label: "Stream ended", dot: IDLE_DOT };
  if (status === "connecting") return { label: "Connecting…", dot: IDLE_DOT };
  if (status === "forbidden") return { label: "Not allowed", dot: IDLE_DOT };
  if (paused) return { label: "Paused", dot: IDLE_DOT };
  if (scrolledUp) return { label: "Scrolled up", dot: IDLE_DOT };
  return { label: "Live", dot: LIVE_DOT };
};

// Reads the scroller: while the view sits at the newest line it follows, and scrolling away releases it.
export const LogStatusMarker = ({ status, reason, paused }: LogStatusMarkerProps) => {
  const { end: scrolledUp } = useMessageScrollerScrollable();
  const { label, dot } = describeMarker(status, paused, scrolledUp);
  return (
    <span role="status" title={reason} className="ml-auto flex items-center gap-2 font-mono text-xs text-muted-foreground">
      <span aria-hidden className={cn("size-1.5 rounded-full motion-reduce:animate-none", dot)} />
      {label}
    </span>
  );
};
