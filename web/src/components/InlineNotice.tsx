import type { ReactNode } from "react";
import { CircleAlert } from "lucide-react";

import { cn } from "@/lib/utils";

interface InlineNoticeProps {
  title: string;
  children?: ReactNode;
  className?: string;
}

// A refusal inside a card or form, left aligned with what it sits above; ErrorDisplay is for a whole page that failed.
export const InlineNotice = ({ title, children, className }: InlineNoticeProps) => (
  <div role="alert" className={cn("flex gap-2.5 rounded-lg border border-destructive/40 px-3 py-2.5 text-[13px] leading-5", className)}>
    <CircleAlert aria-hidden className="mt-0.5 size-4 shrink-0 text-destructive" />
    <div className="min-w-0">
      <p className="font-semibold">{title}</p>
      {children && <div className="text-pretty text-muted-foreground">{children}</div>}
    </div>
  </div>
);
