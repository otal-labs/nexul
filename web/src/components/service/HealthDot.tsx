import { cn } from "@/lib/utils";
import { DeployStatus, type DeployStatus as DeployStatusType } from "@/models/Service";

// Only "healthy" pulses; others stay still since this page is read under stress, where calm color reads clearer.
const dotClass: Record<DeployStatusType, string> = {
  [DeployStatus.Pending]: "bg-warning",
  [DeployStatus.Running]: "bg-info",
  [DeployStatus.Healthy]: "bg-success animate-[status-pulse_2.4s_ease-standard_infinite]",
  [DeployStatus.Failed]: "bg-destructive",
};

interface HealthDotProps {
  status: DeployStatusType;
  className?: string;
}

// Decorative (aria-hidden): the adjacent badge/text carries the label; prefers-reduced-motion collapses the pulse.
export const HealthDot = ({ status, className }: HealthDotProps) => (
  <span
    aria-hidden
    className={cn(
      "size-2 rounded-full transition-colors duration-150 ease-standard",
      dotClass[status],
      className,
    )}
  />
);
