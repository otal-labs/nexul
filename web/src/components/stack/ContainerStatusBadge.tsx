import { CheckCircle2, Clock, Loader2, MinusCircle, XCircle, type LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { ContainerStatus, type ContainerStatus as ContainerStatusType } from "@/models/Stack";

// Colored icon beside plain text, never a filled/tinted chip; semantic tokens carry color instead of background tint.
const config: Record<ContainerStatusType, { icon: LucideIcon; color: string; iconClassName?: string; textClassName?: string }> = {
  [ContainerStatus.Pending]: { icon: Clock, color: "text-warning" },
  [ContainerStatus.Running]: { icon: Loader2, color: "text-info", iconClassName: "animate-spin motion-reduce:animate-none" },
  [ContainerStatus.Healthy]: { icon: CheckCircle2, color: "text-success" },
  [ContainerStatus.Exited]: { icon: XCircle, color: "text-destructive", textClassName: "text-foreground" },
  [ContainerStatus.Stopped]: { icon: MinusCircle, color: "text-muted-foreground" },
};

interface ContainerStatusBadgeProps {
  status: ContainerStatusType;
}

export const ContainerStatusBadge = ({ status }: ContainerStatusBadgeProps) => {
  // The server stores whatever word a runner reports, so one the app doesn't know yet still renders.
  const { icon, color, iconClassName, textClassName } = config[status] ?? config[ContainerStatus.Stopped];
  return (
    <NoFillBadge icon={icon} color={color} iconClassName={iconClassName} className={textClassName}>
      {status}
    </NoFillBadge>
  );
};
