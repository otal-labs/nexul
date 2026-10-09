import { CircleAlert, CircleCheck, CircleDot, CircleX, type LucideIcon } from "lucide-react";

import type { EmbedTone } from "@/models/Embed";

export const toneText: Record<EmbedTone, string> = {
  success: "text-success",
  warning: "text-warning",
  destructive: "text-destructive",
  info: "text-info",
};

export const toneBg: Record<EmbedTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  destructive: "bg-destructive",
  info: "bg-info",
};

export const toneIcon: Record<EmbedTone, LucideIcon> = {
  success: CircleCheck,
  warning: CircleAlert,
  destructive: CircleX,
  info: CircleDot,
};
