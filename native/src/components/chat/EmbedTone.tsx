import { CircleAlert, CircleCheck, CircleDot, CircleX, type LucideIcon } from "lucide-react-native";

import type { EmbedTone } from "@/models/Embed";

export const toneBg: Record<EmbedTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  destructive: "bg-destructive",
  info: "bg-info",
};

export const toneVar: Record<EmbedTone, string> = {
  success: "--color-success",
  warning: "--color-warning",
  destructive: "--color-destructive",
  info: "--color-info",
};

export const toneIcon: Record<EmbedTone, LucideIcon> = {
  success: CircleCheck,
  warning: CircleAlert,
  destructive: CircleX,
  info: CircleDot,
};
