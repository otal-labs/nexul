import { useEffect } from "react";
import Animated, { interpolate, useAnimatedStyle, useSharedValue, withTiming } from "react-native-reanimated";

import { ease, useReducedMotion } from "@/lib/motion";
import { cn } from "@/lib/utils";

interface UnreadDotProps {
  unread: boolean;
  className?: string;
}

// An ink dot while something is unread; once read it shrinks to 0.6 as it fades over 150ms, in step with the title letting go.
export const UnreadDot = ({ unread, className }: UnreadDotProps) => {
  const reduced = useReducedMotion();
  const shown = useSharedValue(unread ? 1 : 0);

  useEffect(() => {
    shown.set(withTiming(unread ? 1 : 0, { duration: 150, easing: ease.standard }));
  }, [shown, unread]);

  const style = useAnimatedStyle(() => ({
    opacity: shown.get(),
    transform: [{ scale: reduced ? 1 : interpolate(shown.get(), [0, 1], [0.6, 1]) }],
  }));
  return (
    <Animated.View
      accessible={unread}
      accessibilityLabel={unread ? "Unread" : undefined}
      style={style}
      className={cn("size-2.5 rounded-full border-2 border-background bg-foreground", className)}
    />
  );
};
