import { CheckCircle2, PauseCircle, PlayCircle, XCircle, type LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { ServiceStatus } from "@/models/Topology";
import { cn } from "@/lib/utils";

const statusIcons: Record<ServiceStatus, LucideIcon> = {
  [ServiceStatus.Healthy]: CheckCircle2,
  [ServiceStatus.Running]: PlayCircle,
  [ServiceStatus.Stopped]: PauseCircle,
  [ServiceStatus.Failed]: XCircle,
};

const statusColors: Record<ServiceStatus, string> = {
  [ServiceStatus.Healthy]: "text-success",
  [ServiceStatus.Running]: "text-info",
  [ServiceStatus.Stopped]: "text-muted-foreground",
  [ServiceStatus.Failed]: "text-destructive",
};

interface StatusBadgeProps {
  status: ServiceStatus;
  className?: string;
}

export const StatusBadge = ({ status, className }: StatusBadgeProps) => (
  <NoFillBadge icon={statusIcons[status]} color={statusColors[status]} className={cn("shrink-0", className)}>
    {status}
  </NoFillBadge>
);
