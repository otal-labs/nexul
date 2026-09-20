import { Circle, CircleCheck, Clock } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { AutomationVersionStatus } from "@/enums/Automation";

interface AutomationVersionStatusBadgeProps {
  status: AutomationVersionStatus;
}

export const AutomationVersionStatusBadge = ({ status }: AutomationVersionStatusBadgeProps) => {
  if (status === AutomationVersionStatus.Active) {
    return (
      <NoFillBadge icon={CircleCheck} color="text-success">
        Active
      </NoFillBadge>
    );
  }
  if (status === AutomationVersionStatus.Pending) {
    return (
      <NoFillBadge icon={Clock} color="text-warning">
        Pending
      </NoFillBadge>
    );
  }
  return (
    <NoFillBadge icon={Circle} color="text-muted-foreground">
      Inactive
    </NoFillBadge>
  );
};
