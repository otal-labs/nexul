import type { ReactNode } from "react";
import { View } from "react-native";

import { avatarFill } from "@/lib/color";
import { cn } from "@/lib/utils";

interface SeededTileProps {
  seed: string;
  className?: string;
  children?: ReactNode;
}

// The flat seeded fill behind a person without a photo and behind a project's mark; the size and radius come from className.
export const SeededTile = ({ seed, className, children }: SeededTileProps) => (
  <View style={{ backgroundColor: avatarFill(seed) }} className={cn("items-center justify-center overflow-hidden", className)}>
    {children}
  </View>
);
