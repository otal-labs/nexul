import CircleAlert from "lucide-react-native/icons/circle-alert";
import CircleCheck from "lucide-react-native/icons/circle-check";
import CircleDot from "lucide-react-native/icons/circle-dot";
import CircleX from "lucide-react-native/icons/circle-x";
import type { LucideIcon } from "lucide-react-native";

import type { EmbedTone } from "@nexul/client-core/embed";

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
