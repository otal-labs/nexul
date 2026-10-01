import { motion, useReducedMotion } from "motion/react";
import type { Variants } from "motion/react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type ToolRowStatus = "running" | "failed" | "idle";

interface ToolRowProps {
  // icon is absent for the Agent's own sentence, which reads as prose from the icon column.
  icon?: ReactNode;
  // label is the row's one line, truncated; mono for a command, a path, or arguments, plain for a name or a sentence.
  label: string;
  mono?: boolean;
  status: ToolRowStatus;
  index?: number;
  trailing?: ReactNode | undefined;
  // entrance plays the mount animation; off for rows re-mounted by expanding a finished turn.
  entrance?: boolean;
}

const STAGGER_ROWS = 5;

const rowVariants: Variants = {
  hidden: { opacity: 0, y: 4 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { duration: 0.2, ease: [0.16, 1, 0.3, 1], delay: Math.min(i, STAGGER_ROWS) * 0.03 },
  }),
};

// One action as a flat line: a bare muted icon, the label, a spinner while it runs, and a fixed slot for the expand
// chevron so every label ends at the same edge. Color only on a failed icon and the spinner.
export const ToolRow = ({ icon, label, mono = false, status, index = 0, trailing, entrance = true }: ToolRowProps) => {
  const reduced = useReducedMotion();
  const still = reduced || !entrance;
  const failed = status === "failed";

  return (
    <motion.div
      className="flex min-h-7 items-center gap-1.5 rounded-md px-0.5 py-0.5 transition-colors duration-150 ease-standard hover:bg-accent/40"
      variants={rowVariants}
      custom={index}
      initial={still ? false : "hidden"}
      animate="visible"
    >
      {icon && (
        <span
          className={cn("flex size-6 shrink-0 items-center justify-center", failed ? "text-destructive" : "text-muted-foreground")}
          role={failed ? "img" : undefined}
          aria-label={failed ? "failed" : undefined}
        >
          {icon}
        </span>
      )}
      <span
        className={cn(
          "min-w-0 flex-1 text-sm leading-relaxed",
          icon ? "truncate text-muted-foreground" : "px-1 text-foreground/80",
          mono && "font-mono text-xs",
        )}
      >
        {label}
      </span>
      {status === "running" && (
        <span
          className="block size-3 shrink-0 animate-spin motion-reduce:animate-none rounded-full border-[1.5px] border-warning/25 border-t-warning"
          role="img"
          aria-label="running"
        />
      )}
      <span className="flex size-4 shrink-0 items-center justify-center">{trailing}</span>
    </motion.div>
  );
};
