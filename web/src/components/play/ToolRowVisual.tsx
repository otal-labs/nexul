import { Check, X } from "lucide-react";
import { motion, useReducedMotion } from "motion/react";
import type { Variants } from "motion/react";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export type ToolRowStatus = "pending" | "running" | "done" | "failed" | "idle";

interface ToolRowProps {
  icon: ReactNode;
  // label is the row's one line, truncated; mono for a tool call, a command, or a path, plain for a sentence.
  label: string;
  mono?: boolean;
  result?: string | undefined;
  meta?: string | undefined;
  status: ToolRowStatus;
  index?: number;
  trailing?: ReactNode | undefined;
  // entrance plays the mount animation; off for rows re-mounted by expanding a finished turn.
  entrance?: boolean;
  className?: string;
}

const STAGGER_ROWS = 5;
const STATUS_FADE = { duration: 0.3, ease: "easeOut" as const };

const rowVariants: Variants = {
  hidden: { opacity: 0, y: 8 },
  visible: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { type: "spring", stiffness: 320, damping: 26, delay: Math.min(i, STAGGER_ROWS) * 0.1 },
  }),
};

// One action row: bordered icon square, one truncated line, and a status slot that is a dot while pending, a ring
// spinner while running, and the meta plus a check (or cross) once done. Color only in the status slot.
export const ToolRow = ({ icon, label, mono = false, result, meta, status, index = 0, trailing, entrance = true, className }: ToolRowProps) => {
  const reduced = useReducedMotion();
  const still = reduced || !entrance;
  const pending = status === "pending";
  const running = status === "running";
  const settled = status === "done" || status === "failed";

  return (
    <motion.div
      className={cn("relative flex items-center gap-2.5 rounded-lg px-2 py-1", className)}
      variants={rowVariants}
      custom={index}
      initial={still ? false : "hidden"}
      animate="visible"
    >
      <motion.span
        className="pointer-events-none absolute inset-0 rounded-lg bg-primary/5 ring-1 ring-primary/20 ring-inset"
        initial={false}
        animate={{ opacity: running ? 1 : 0 }}
        transition={STATUS_FADE}
      />

      <span
        className={cn(
          "relative flex size-7 shrink-0 items-center justify-center rounded-lg border bg-card transition-colors duration-300 ease-standard",
          pending && "text-muted-foreground",
          !pending && "text-foreground",
        )}
      >
        {icon}
      </span>

      <span className="relative flex min-w-0 flex-1 items-center gap-1.5">
        <span className={cn("min-w-0 truncate text-sm text-foreground", mono && "font-mono text-[11px]")}>{label}</span>
        {settled && result && (
          <motion.span
            className="flex min-w-0 shrink items-center gap-1 truncate font-mono text-[10px] text-muted-foreground"
            initial={still ? false : { opacity: 0, x: -4 }}
            animate={{ opacity: 1, x: 0 }}
            transition={STATUS_FADE}
          >
            <span className="text-muted-foreground/40">→</span>
            <span className="truncate text-foreground/70">{result}</span>
          </motion.span>
        )}
      </span>

      <span className="relative flex h-4 shrink-0 items-center justify-end gap-1.5">
        {meta && <span className="font-mono text-[10px] text-muted-foreground tabular-nums">{meta}</span>}
        {pending && <span className="block size-1.5 rounded-full bg-muted-foreground/40" aria-hidden />}
        {running && (
          <span
            className="block size-3.5 animate-spin rounded-full border-[1.5px] border-warning/25 border-t-warning"
            role="img"
            aria-label="running"
          />
        )}
        {settled && (
          <motion.span
            className="flex items-center"
            initial={still ? false : { opacity: 0, scale: 0.7 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ type: "spring", stiffness: 420, damping: 18 }}
          >
            {status === "done" && <Check className="size-3.5 text-success" strokeWidth={3} role="img" aria-label="done" />}
            {status === "failed" && <X className="size-3.5 text-destructive" strokeWidth={3} role="img" aria-label="failed" />}
          </motion.span>
        )}
        {trailing}
      </span>
    </motion.div>
  );
};
