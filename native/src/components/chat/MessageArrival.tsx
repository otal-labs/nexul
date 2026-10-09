import { useState, type ReactNode } from "react";
import Animated, { interpolate, useAnimatedStyle } from "react-native-reanimated";

import { useEntrance, useReducedMotion } from "@/lib/motion";

// "rise" is your own message sent from here; "arrive" is anyone else's landing while the thread is open.
export type Arrival = "rise" | "arrive";

interface MessageArrivalProps {
  arrival: Arrival | undefined;
  children: ReactNode;
}

// Your message rises 12px over 240ms while its bubble grows from 0.96 out of its bottom-right corner; someone else's rises
// 8px over 200ms. Nothing waits on it. Under reduced motion both are a 150ms fade.
const Arriving = ({ arrival, children }: { arrival: Arrival; children: ReactNode }) => {
  const reduced = useReducedMotion();
  const progress = useEntrance(reduced ? 150 : arrival === "rise" ? 240 : 200);
  const travel = arrival === "rise" ? 12 : 8;
  const grow = arrival === "rise" ? 0.96 : 1;
  const style = useAnimatedStyle(() => ({
    opacity: progress.get(),
    transform: [
      { translateY: reduced ? 0 : interpolate(progress.get(), [0, 1], [travel, 0]) },
      { scale: reduced ? 1 : interpolate(progress.get(), [0, 1], [grow, 1]) },
    ],
  }));
  return <Animated.View style={[style, { transformOrigin: "right bottom" }]}>{children}</Animated.View>;
};

// What was on screen at open never moves, so it mounts as the bare row: a thread opens on a screenful of them at once.
export const MessageArrival = ({ arrival: requested, children }: MessageArrivalProps) => {
  // Latched at mount: the confirmed copy that takes over a pending row keeps its rise running.
  const [arrival] = useState(requested);
  return (
    <>
      {arrival && <Arriving arrival={arrival}>{children}</Arriving>}
      {!arrival && children}
    </>
  );
};
