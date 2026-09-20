import { CheckCircle2, Clock, Loader2, MinusCircle, XCircle, type LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { ContainerStatus, type ContainerStatus as ContainerStatusType } from "@/models/Stack";

// Colored icon + text, never a filled/tinted chip; semantic tokens carry color instead of background tint.
const config: Record<ContainerStatusType, { icon: LucideIcon; color: string }> = {
  [ContainerStatus.Pending]: { icon: Clock, color: "text-warning" },
  [ContainerStatus.Running]: { icon: Loader2, color: "text-info" },
  [ContainerStatus.Healthy]: { icon: CheckCircle2, color: "text-success" },
  [ContainerStatus.Exited]: { icon: XCircle, color: "text-destructive" },
  [ContainerStatus.Stopped]: { icon: MinusCircle, color: "text-muted-foreground" },
};

interface ContainerStatusBadgeProps {
  status: ContainerStatusType;
}

export const ContainerStatusBadge = ({ status }: ContainerStatusBadgeProps) => {
  const { icon, color } = config[status];
  return (
    <NoFillBadge icon={icon} color={color}>
      {status}
    </NoFillBadge>
  );
};
