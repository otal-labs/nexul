import { CheckCircle2, Clock, Loader2, XCircle, type LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { DeployStatus, type DeployStatus as DeployStatusType } from "@/models/Service";

// Colored icon + text, never a filled/tinted chip; semantic tokens carry color instead of background tint.
const config: Record<DeployStatusType, { icon: LucideIcon; color: string }> = {
  [DeployStatus.Pending]: { icon: Clock, color: "text-warning" },
  [DeployStatus.Running]: { icon: Loader2, color: "text-info" },
  [DeployStatus.Healthy]: { icon: CheckCircle2, color: "text-success" },
  [DeployStatus.Failed]: { icon: XCircle, color: "text-destructive" },
};

interface DeployStatusBadgeProps {
  status: DeployStatusType;
}

export const DeployStatusBadge = ({ status }: DeployStatusBadgeProps) => {
  const { icon, color } = config[status];
  return (
    <NoFillBadge icon={icon} color={color}>
      {status}
    </NoFillBadge>
  );
};
