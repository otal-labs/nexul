import type { ReactNode } from "react";
import { CircleAlert, Info } from "lucide-react";

import { cn } from "@/lib/utils";

interface InlineNoticeProps {
  title: string;
  children?: ReactNode;
  // "info" says what just happened without calling it a failure.
  tone?: "error" | "info";
  className?: string;
}

// A refusal or a note inside a card or form, left aligned with what it sits above; ErrorDisplay is for a whole page that failed.
export const InlineNotice = ({ title, children, tone = "error", className }: InlineNoticeProps) => {
  const Icon = tone === "error" ? CircleAlert : Info;
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={cn("flex gap-2.5 rounded-lg border px-3 py-2.5 text-[13px] leading-5", tone === "error" ? "border-destructive/40" : "border-border", className)}
    >
      <Icon aria-hidden className={cn("mt-0.5 size-4 shrink-0", tone === "error" ? "text-destructive" : "text-muted-foreground")} />
      <div className="min-w-0">
        <p className="font-semibold">{title}</p>
        {children && <div className="text-pretty text-muted-foreground">{children}</div>}
      </div>
    </div>
  );
};
