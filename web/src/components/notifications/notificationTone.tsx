import { CircleCheckIcon, CircleIcon, CircleStopIcon, CircleXIcon, type LucideIcon } from "lucide-react";

import { notificationKindIcon } from "@/components/notifications/notificationKindIcon";
import { NotificationKind, type Notification } from "@/models/Notification";
import { playRunOutcome, type PlayRunOutcome } from "@/utils/InboxUtility";

// The notices that carry a state draw their kind icon in its hue; every other kind stays muted.
export const notificationToneText: Partial<Record<NotificationKind, string>> = {
  [NotificationKind.ServiceStopped]: "text-destructive",
  [NotificationKind.ErrorRateCrossed]: "text-destructive",
  [NotificationKind.ServiceRecovered]: "text-success",
  [NotificationKind.ErrorRateRecovered]: "text-success",
  [NotificationKind.PlayRunWaiting]: "text-warning",
  [NotificationKind.TicketDueSoon]: "text-warning",
};

const runMarks: Record<PlayRunOutcome, { icon: LucideIcon; tone: string; label: string }> = {
  finished: { icon: CircleCheckIcon, tone: "text-success", label: "Run finished" },
  failed: { icon: CircleXIcon, tone: "text-destructive", label: "Run failed" },
  interrupted: { icon: CircleStopIcon, tone: "text-warning", label: "Run interrupted" },
};

/** The icon a row leads with, in its status hue when the notice carries a state; a play run shows how it ended. */
export const notificationMark = (n: Notification): { icon: LucideIcon; tone: string | undefined; label: string | undefined } => {
  if (n.kind === NotificationKind.PlayRunFinished) return runMarks[playRunOutcome(n)];
  return { icon: notificationKindIcon[n.kind] ?? CircleIcon, tone: notificationToneText[n.kind], label: undefined };
};
