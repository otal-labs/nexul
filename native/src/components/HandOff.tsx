import { useEffect, useState, type ReactNode } from "react";
import Animated, { interpolate, useAnimatedStyle } from "react-native-reanimated";

import { LOADER_DELAY_MS } from "@/components/LoadingDisplay";
import { useEntrance, useReducedMotion } from "@/lib/motion";

// True once a load has run long enough for its loader to have shown; a cached screen never waits, so never hands off.
export const useLoaderShown = (isPending: boolean) => {
  const [shown, setShown] = useState(false);
  useEffect(() => {
    if (!isPending) return;
    const timer = setTimeout(() => setShown(true), LOADER_DELAY_MS);
    return () => clearTimeout(timer);
  }, [isPending]);
  return shown;
};

interface HandOffProps {
  // Whether a loader stood here; content that replaces one rises 6px and fades in over 200ms where it stood.
  after: boolean;
  children: ReactNode;
}

// Late data arrives the way the web's does after its loader: the page entrance's rise, or a 150ms fade under reduced motion.
export const HandOff = ({ after, children }: HandOffProps) => {
  const reduced = useReducedMotion();
  const [animate] = useState(after);
  const progress = useEntrance(reduced ? 150 : 200, 0, !animate);
  const style = useAnimatedStyle(() => ({
    opacity: progress.get(),
    transform: [{ translateY: reduced ? 0 : interpolate(progress.get(), [0, 1], [6, 0]) }],
  }));
  return <Animated.View style={[{ flex: 1 }, style]}>{children}</Animated.View>;
};
