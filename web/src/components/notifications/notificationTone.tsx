import { NotificationKind } from "@/models/Notification";

// The notices that carry a state draw their kind icon in its hue; every other kind stays muted.
export const notificationToneText: Partial<Record<NotificationKind, string>> = {
  [NotificationKind.ServiceStopped]: "text-destructive",
  [NotificationKind.ErrorRateCrossed]: "text-destructive",
  [NotificationKind.ServiceRecovered]: "text-success",
  [NotificationKind.ErrorRateRecovered]: "text-success",
  [NotificationKind.PlayRunWaiting]: "text-warning",
  [NotificationKind.TicketDueSoon]: "text-warning",
};
