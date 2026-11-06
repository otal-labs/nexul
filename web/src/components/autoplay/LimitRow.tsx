import type { ReactNode } from "react";

interface LimitRowProps {
  label: ReactNode;
  hint?: string;
  children: ReactNode;
}

// One limit: what it bounds on the left, its control on the block's right edge, centred on the label's line even with a hint under it.
export const LimitRow = ({ label, hint, children }: LimitRowProps) => (
  <div className="flex items-start gap-4 py-2.5">
    <div className="min-w-0 flex-1">
      <div className="truncate text-row text-foreground">{label}</div>
      {hint && <div className="text-xs text-muted-foreground">{hint}</div>}
    </div>
    <div className="flex h-5 shrink-0 items-center">{children}</div>
  </div>
);
