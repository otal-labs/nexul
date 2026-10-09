import type { LucideIcon } from "lucide-react-native";
import { View } from "react-native";
import { useCSSVariable } from "uniwind";

import { cn } from "@/lib/utils";

interface StatusTileProps {
  icon: LucideIcon;
  // A status dot class (bg-success, bg-warning…); the dot sits on the tile's corner, cut out by a canvas ring.
  dot?: string | undefined;
  dashed?: boolean;
}

// The leading mark of a list row: a muted glyph on a card tile, its state as a dot on the corner.
export const StatusTile = ({ icon: Icon, dot, dashed = false }: StatusTileProps) => {
  const [muted] = useCSSVariable(["--color-muted-foreground"]);
  return (
    <View className={cn("size-9 items-center justify-center rounded-lg border", dashed ? "border-dashed border-input" : "border-border bg-card")}>
      <Icon size={16} color={String(muted)} />
      {dot && <View className={cn("absolute -right-1 -top-1 size-3 rounded-full border-2 border-background", dot)} />}
    </View>
  );
};
