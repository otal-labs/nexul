import { CheckCircle2Icon, CircleIcon, ClockIcon, XCircleIcon } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import { NoFillBadge } from "@/components/ui/badge";
import { TicketStatus, type TicketStatus as TicketStatusType } from "@/models/Ticket";

const icons: Record<TicketStatusType, LucideIcon> = {
  [TicketStatus.Open]: CircleIcon,
  [TicketStatus.InProgress]: ClockIcon,
  [TicketStatus.Done]: CheckCircle2Icon,
  [TicketStatus.Closed]: XCircleIcon,
};

const colors: Record<TicketStatusType, string> = {
  [TicketStatus.Open]: "text-info",
  [TicketStatus.InProgress]: "text-warning",
  [TicketStatus.Done]: "text-success",
  [TicketStatus.Closed]: "text-muted-foreground",
};

interface TicketStatusBadgeProps {
  status: TicketStatusType;
}

export const TicketStatusBadge = ({ status }: TicketStatusBadgeProps) => (
  <NoFillBadge icon={icons[status]} color={colors[status]}>
    {status.replace("_", " ")}
  </NoFillBadge>
);
