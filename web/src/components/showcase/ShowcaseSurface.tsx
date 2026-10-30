import type { ReactNode } from "react";

import { ShowcaseField } from "@/components/showcase/ShowcaseField";
import { useSessionStore } from "@/stores/sessionStore";
import { cn } from "@/lib/utils";

interface ShowcaseSurfaceProps {
  live?: boolean;
  quiet?: boolean;
  // Fills the screen even signed in, for a page drawn outside the app frame.
  fullScreen?: boolean;
  className?: string;
  children: ReactNode;
}

// Signed out it fills the screen; signed in it takes the frame's place as one opaque surface, so nothing moves behind a blur.
export const ShowcaseSurface = ({ live = false, quiet = false, fullScreen = false, className, children }: ShowcaseSurfaceProps) => {
  const framed = useSessionStore((s) => s.isLoggedIn) && !fullScreen;
  return (
    <div
      data-pane-layout={framed ? "" : undefined}
      className={cn(
        "relative isolate overflow-hidden",
        framed
          ? "h-full rounded-xl border border-transparent border-t-(--panel-highlight) shadow-[0_0_0_1px_var(--panel-ring),var(--panel-shadow)]"
          : "h-dvh",
      )}
    >
      <ShowcaseField live={live} quiet={quiet} className="-z-10" />
      <div className={cn("relative h-full overflow-y-auto", className)}>{children}</div>
    </div>
  );
};
