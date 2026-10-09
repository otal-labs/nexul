import { View } from "react-native";
import Animated, { useAnimatedStyle } from "react-native-reanimated";
import Svg, { Circle } from "react-native-svg";
import { useCSSVariable } from "uniwind";

import { Text } from "@/components/ui/text";
import { useEntrance, useReducedMotion } from "@/lib/motion";

interface LoadingDisplayProps {
  message?: string;
}

// A keyframe loop, so the turn runs on the UI thread with no JavaScript per frame.
const SPIN = {
  animationName: { to: { transform: [{ rotate: "360deg" }] } },
  animationDuration: "1.2s",
  animationTimingFunction: "linear",
  animationIterationCount: "infinite",
} as const;

// Held back so a fast load never flashes a loader.
export const LOADER_DELAY_MS = 300;

// The empty state's orbit at spinner size: an ember dot circling a faint ring, a linear 1.2s turn, still under reduced motion.
export const LoadingDisplay = ({ message = "Loading" }: LoadingDisplayProps) => {
  const [muted, brand] = useCSSVariable(["--color-muted-foreground", "--color-brand"]);
  const reduced = useReducedMotion();
  const shown = useEntrance(150, LOADER_DELAY_MS);
  const fade = useAnimatedStyle(() => ({ opacity: shown.get() }));
  const spin = reduced ? undefined : SPIN;
  return (
    <Animated.View accessible role="progressbar" aria-label={message} style={fade} className="flex-1 flex-row items-center justify-center gap-2.5 px-6 py-10">
      <Animated.View style={spin}>
        <Svg width={16} height={16} viewBox="0 0 20 20">
          <Circle cx={10} cy={10} r={7.5} stroke={String(muted)} strokeOpacity={0.35} strokeWidth={1.5} fill="none" />
          <Circle cx={10} cy={2.5} r={2.25} fill={String(brand)} />
        </Svg>
      </Animated.View>
      <View>
        <Text className="text-sm text-muted-foreground">{message}</Text>
      </View>
    </Animated.View>
  );
};
