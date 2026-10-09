import type { ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import Svg, { Defs, LinearGradient, Rect, Stop } from "react-native-svg";

import { avatarGradient } from "@/lib/color";
import { cn } from "@/lib/utils";

interface GradientTileProps {
  seed: string;
  className?: string;
  children?: ReactNode;
}

// The seeded gradient behind a person without a photo and behind a project's mark; the size and radius come from className.
export const GradientTile = ({ seed, className, children }: GradientTileProps) => {
  const { turn, stops } = avatarGradient(seed);
  const rad = (turn * Math.PI) / 180;
  const x = Math.cos(rad) / 2;
  const y = Math.sin(rad) / 2;
  return (
    <View className={cn("items-center justify-center overflow-hidden", className)}>
      <Svg style={StyleSheet.absoluteFill} width="100%" height="100%">
        <Defs>
          <LinearGradient id="g" x1={0.5 - x} y1={0.5 - y} x2={0.5 + x} y2={0.5 + y}>
            <Stop offset="0" stopColor={stops[0]} />
            <Stop offset="0.5" stopColor={stops[1]} />
            <Stop offset="1" stopColor={stops[2]} />
          </LinearGradient>
        </Defs>
        <Rect width="100%" height="100%" fill="url(#g)" />
      </Svg>
      {children}
    </View>
  );
};
