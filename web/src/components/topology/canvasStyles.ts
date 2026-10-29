import { ServiceStatus, type AttentionStatus } from "@/models/Topology";

// A service card's two ways in: route wires land on the left, hand-drawn relations on the right.
export const ROUTE_IN = "route-in";
export const RELATION_IN = "relation-in";

// A route wire's handle: invisible and a pixel wide, so the wire ends on the card's edge instead of short of it.
export const wireHandle = "!size-px !min-h-0 !min-w-0 !border-0 !opacity-0";

// Handles show only on hover, where a wire can start; at rest a ring of dots around every card is noise.
export const handleClasses =
  "!bg-muted-foreground !opacity-0 !transition-opacity !duration-150 !ease-standard group-hover:!opacity-100";

// Every card on the canvas lifts 2px on hover and takes the brand ring when selected.
export const nodeShell =
  "group canvas-card transition-[translate] duration-150 ease-standard hover:-translate-y-0.5 data-[selected=true]:outline-2 data-[selected=true]:outline-ring";

export const statusDot: Record<ServiceStatus, string> = {
  [ServiceStatus.Healthy]: "bg-success",
  [ServiceStatus.Running]: "bg-info",
  [ServiceStatus.Stopped]: "bg-warning",
  [ServiceStatus.Failed]: "bg-destructive",
};

export const toneText: Record<AttentionStatus, string> = {
  [ServiceStatus.Failed]: "text-destructive",
  [ServiceStatus.Stopped]: "text-warning",
};

export const wireStroke = (tone: AttentionStatus | undefined) => {
  if (tone === ServiceStatus.Failed) return "var(--destructive)";
  if (tone === ServiceStatus.Stopped) return "var(--warning)";
  return "var(--muted-foreground)";
};
