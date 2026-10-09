import { motion, useReducedMotion, type HTMLMotionProps } from "motion/react";

import { EASE_OUT_CURVE, ROW_GLIDE } from "@/lib/motion";

interface MotionRowProps extends HTMLMotionProps<"li"> {
  // The row's place in its list: rows glide only when it changes, so a neighbour growing open still just snaps.
  index: number;
}

// A list row that leaves and moves. Removed, it fades and settles to 0.98 where it stood while the rows after it
// glide up into the gap; the list wraps its rows in AnimatePresence with mode="popLayout". Arrivals stay EnterList's.
export const MotionRow = ({ index, ...props }: MotionRowProps) => {
  const reduced = useReducedMotion() ?? false;
  return (
    <motion.li
      layout={reduced ? false : "position"}
      layoutDependency={index}
      exit={reduced ? { opacity: 0 } : { opacity: 0, scale: 0.98 }}
      transition={{ layout: ROW_GLIDE, default: { duration: 0.12, ease: EASE_OUT_CURVE } }}
      {...props}
    />
  );
};
