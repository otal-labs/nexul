import type { ReactNode } from "react";

import { ShowcaseField } from "@/components/showcase/ShowcaseField";
import { useSessionStore } from "@/stores/sessionStore";
import { cn } from "@/lib/utils";

interface ShowcaseSurfaceProps {
  live?: boolean;
  quiet?: boolean;
  className?: string;
  children: ReactNode;
}

// Signed out it fills the screen; signed in it takes the frame's place as one opaque surface, so nothing moves behind a blur.
export const ShowcaseSurface = ({ live = false, quiet = false, className, children }: ShowcaseSurfaceProps) => {
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  return (
    <div
      data-pane-layout={isLoggedIn ? "" : undefined}
      className={cn(
        "relative isolate overflow-hidden",
        isLoggedIn
          ? "h-full rounded-xl border border-transparent shadow-[inset_0_1px_0_0_var(--panel-highlight),0_0_0_1px_var(--panel-ring),var(--panel-shadow)]"
          : "h-dvh",
      )}
    >
      <ShowcaseField live={live} quiet={quiet} className="-z-10" />
      <div className={cn("relative h-full overflow-y-auto", className)}>{children}</div>
    </div>
  );
};
