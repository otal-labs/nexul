import {
  Circle,
  CircleCheckBig,
  CircleDashed,
  CircleDot,
  CircleEllipsis,
  CircleX,
  type LucideIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { STATUS_ICON_NAMES, statusStage, type BoardStatus, type StatusIconName } from "@/models/Status";

const statusIcons: Record<StatusIconName, LucideIcon> = {
  CircleDashed,
  Circle,
  CircleDot,
  CircleEllipsis,
  CircleCheckBig,
  CircleX,
};

export const isStatusIconName = (icon: string): icon is StatusIconName =>
  (STATUS_ICON_NAMES as readonly string[]).includes(icon);

interface StatusIconProps {
  icon: string;
  className?: string;
}

export const StatusIcon = ({ icon, className }: StatusIconProps) => {
  if (!isStatusIconName(icon)) return null;
  const Icon = statusIcons[icon];
  return <Icon className={className} aria-hidden />;
};

interface StatusMarkProps {
  status: Pick<BoardStatus, "kind" | "icon">;
  className?: string;
}

export const StatusMark = ({ status, className }: StatusMarkProps) => {
  const stage = statusStage(status.kind);
  const hasIcon = isStatusIconName(status.icon);
  return (
    <>
      {hasIcon && <StatusIcon icon={status.icon} className={cn("shrink-0", stage.text, className)} />}
      {!hasIcon && <span className={cn("size-2 shrink-0 rounded-full", stage.dot)} aria-hidden />}
    </>
  );
};
