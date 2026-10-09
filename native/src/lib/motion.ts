import { useEffect, useSyncExternalStore } from "react";
import { AccessibilityInfo } from "react-native";
import { Easing, useSharedValue, withDelay, withTiming, type SharedValue } from "react-native-reanimated";

// The web's curves (web/src/index.css), so a moment reads the same on both.
export const ease = {
  out: Easing.bezier(0.16, 1, 0.3, 1),
  standard: Easing.bezier(0.25, 0.1, 0.25, 1),
};

let reduced = false;
const listeners = new Set<() => void>();
const publish = (value: boolean) => {
  reduced = value;
  listeners.forEach((notify) => notify());
};

void AccessibilityInfo.isReduceMotionEnabled().then(publish);
AccessibilityInfo.addEventListener("reduceMotionChanged", publish);

const subscribe = (notify: () => void) => {
  listeners.add(notify);
  return () => listeners.delete(notify);
};

// Android's "Remove animations", live: every moment keeps a short fade and nothing travels, scales or loops.
export const useReducedMotion = () => useSyncExternalStore(subscribe, () => reduced);

// 0 → 1 once on mount over the duration on the out curve, after the delay; 1 from the start when still.
export const useEntrance = (duration: number, delay = 0, still = false): SharedValue<number> => {
  const progress = useSharedValue(still ? 1 : 0);
  useEffect(() => {
    if (still) return;
    progress.set(withDelay(delay, withTiming(1, { duration, easing: ease.out })));
  }, [progress, duration, delay, still]);
  return progress;
};
