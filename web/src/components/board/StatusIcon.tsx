import {
  Circle,
  CircleCheckBig,
  CircleDashed,
  CircleDot,
  CircleEllipsis,
  CircleX,
  type LucideIcon,
} from "lucide-react";

import { STATUS_ICON_NAMES, type StatusIconName } from "@/models/Status";

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
