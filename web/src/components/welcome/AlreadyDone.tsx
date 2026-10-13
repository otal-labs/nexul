import { Check } from "lucide-react";
import type { ReactNode } from "react";

interface AlreadyDoneProps {
  title: ReactNode;
  children: ReactNode;
}

// A step that was done when the wizard reached it says what it found, and its footer only moves on.
export const AlreadyDone = ({ title, children }: AlreadyDoneProps) => (
  <div className="flex items-start gap-3 rounded-lg border border-border bg-card px-4 py-3">
    <span className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-full border border-border text-success">
      <Check className="pop-in size-3" strokeWidth={3} aria-hidden />
    </span>
    <div className="min-w-0 space-y-0.5 text-sm">
      <p className="font-medium">{title}</p>
      <p className="text-muted-foreground">{children}</p>
    </div>
  </div>
);
