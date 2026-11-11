import type { ReactNode } from "react";
import { CircleAlert, Info, TriangleAlert } from "lucide-react";

import { cn } from "@/lib/utils";

interface InlineNoticeProps {
  title: string;
  children?: ReactNode;
  // "info" says what just happened without calling it a failure; "warning" says something will go wrong later.
  tone?: "error" | "info" | "warning";
  className?: string;
}

const tones = {
  error: { Icon: CircleAlert, role: "alert", border: "border-destructive/40", icon: "text-destructive" },
  warning: { Icon: TriangleAlert, role: "status", border: "border-warning/50", icon: "text-warning" },
  info: { Icon: Info, role: "status", border: "border-border", icon: "text-muted-foreground" },
} as const;

// A refusal or a note inside a card or form, left aligned with what it sits above; ErrorDisplay is for a whole page that failed.
export const InlineNotice = ({ title, children, tone = "error", className }: InlineNoticeProps) => {
  const { Icon, role, border, icon } = tones[tone];
  return (
    <div role={role} className={cn("flex gap-2.5 rounded-lg border px-3 py-2.5 text-row leading-5", border, className)}>
      <Icon aria-hidden className={cn("mt-0.5 size-4 shrink-0", icon)} />
      <div className="min-w-0">
        <p className="font-semibold">{title}</p>
        {children && <div className="text-pretty text-muted-foreground">{children}</div>}
      </div>
    </div>
  );
};
