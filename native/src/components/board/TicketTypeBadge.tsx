import Bug from "lucide-react-native/icons/bug";
import FileText from "lucide-react-native/icons/file-text";
import FlaskConical from "lucide-react-native/icons/flask-conical";
import ListTodo from "lucide-react-native/icons/list-todo";
import Milestone from "lucide-react-native/icons/milestone";
import Palette from "lucide-react-native/icons/palette";
import Shapes from "lucide-react-native/icons/shapes";
import Sparkles from "lucide-react-native/icons/sparkles";
import Wrench from "lucide-react-native/icons/wrench";
import type { LucideIcon } from "lucide-react-native";
import { View } from "react-native";

import { hexIconColor, ticketTypeColor } from "@/components/board/ticketTypeColor";
import { Icon } from "@/components/ui/icon";
import { Text } from "@/components/ui/text";

// The web's icon set, inferred from the type name with a generic fallback.
const TYPE_ICONS: Record<string, LucideIcon> = {
  task: ListTodo,
  bug: Bug,
  feature: Sparkles,
  chore: Wrench,
  docs: FileText,
  documentation: FileText,
  epic: Milestone,
  spike: FlaskConical,
  design: Palette,
};

// A type is its icon in the type hue beside plain muted text, never a filled chip.
export const TicketTypeBadge = ({ name, color }: { name: string; color: string | undefined }) => (
  <View className="flex-row items-center gap-1">
    <Icon as={TYPE_ICONS[name.trim().toLowerCase()] ?? Shapes} className={`size-3 ${ticketTypeColor(name, color)}`} {...hexIconColor(color)} />
    <Text numberOfLines={1} className="shrink text-xs font-medium text-muted-foreground">
      {name}
    </Text>
  </View>
);
