import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";

// The dialog's footer tray, opaque like the rest of the body.
export const footerTrayClass = "flex h-14 shrink-0 items-baseline gap-2 border-t border-border bg-[color-mix(in_oklab,var(--surface-2)_60%,var(--popover))] pt-2.5 pr-4 pl-6";

interface HistoryFooterProps {
  /** What the action does, in one line. */
  note: string;
  /** The primary action for the picked version, after Close. */
  children?: ReactNode;
}

export const HistoryFooter = ({ note, children }: HistoryFooterProps) => (
  <div className={footerTrayClass}>
    <p className="min-w-0 flex-1 truncate text-[13px] text-muted-foreground">{note}</p>
    <DialogClose asChild>
      <Button variant="ghost">Close</Button>
    </DialogClose>
    {children}
  </div>
);
