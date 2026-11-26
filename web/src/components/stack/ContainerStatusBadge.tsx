import { Circle } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { ContainerStatus, type ContainerStatus as ContainerStatusType } from "@/models/Stack";

// One still dot in the status hue beside the word, so every state of a service reads as one family of marks.
const config: Record<ContainerStatusType, { color: string; textClassName?: string }> = {
  [ContainerStatus.Pending]: { color: "text-warning-mark" },
  [ContainerStatus.Running]: { color: "text-success" },
  [ContainerStatus.Healthy]: { color: "text-success" },
  [ContainerStatus.Exited]: { color: "text-destructive", textClassName: "text-foreground" },
  [ContainerStatus.Stopped]: { color: "text-muted-foreground" },
};

interface ContainerStatusBadgeProps {
  status: ContainerStatusType;
}

export const ContainerStatusBadge = ({ status }: ContainerStatusBadgeProps) => {
  // The server stores whatever word a runner reports, so one the app doesn't know yet still renders.
  const { color, textClassName } = config[status] ?? config[ContainerStatus.Stopped];
  return (
    <NoFillBadge icon={Circle} color={color} iconClassName="fill-current p-0.5" className={textClassName}>
      {status}
    </NoFillBadge>
  );
};
