import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

// The invite page's one card, centred on the showcase field, for the invitation and its refusals alike.
export const InvitationPanel = ({ children, className }: { children: ReactNode; className?: string }) => (
  <div className="flex min-h-full items-center justify-center px-4 py-10 sm:px-6 sm:py-12">
    <div className={cn("panel w-full max-w-[28rem] p-6 [--panel-opacity:72%] sm:p-8", className)}>{children}</div>
  </div>
);
