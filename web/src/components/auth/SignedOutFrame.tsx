import type { ReactNode } from "react";

import { Logo } from "@/components/Logo";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { cn } from "@/lib/utils";

interface SignedOutFrameProps {
  live?: boolean;
  quiet?: boolean;
  // One way out, top right, where the screen has one.
  exit?: ReactNode;
  // "card" puts the content on the frosted panel; setup's tall forms sit on the field.
  card?: boolean;
  children: ReactNode;
}

// Sign-in, invitation and setup share this frame: the instance in the top-left corner, the screen in one 448px column.
// Setup sits at one height on every step, so a step or tab that grows never moves the header or the control just pressed.
export const SignedOutFrame = ({ live = false, quiet = false, exit, card = false, children }: SignedOutFrameProps) => (
  <ShowcaseSurface live={live} quiet={quiet} fullScreen className="[scrollbar-gutter:stable_both-edges]">
    <div className="flex min-h-full flex-col">
      <div className="flex items-center justify-between gap-4 px-6 pt-6 lg:px-8">
        <span className="flex min-w-0 items-center gap-2">
          <Logo className="size-5 rounded-sm [&_svg]:size-3.5" />
          <span className="truncate font-mono text-xs text-muted-foreground">{window.location.host}</span>
        </span>
        {exit}
      </div>
      <div className={cn("flex flex-1 justify-center px-6 py-5", card ? "items-center" : "items-start pt-[max(1.25rem,6vh)]")}>
        <div className={cn("w-full max-w-md", card && "panel px-8 py-5 [--panel-opacity:72%]")}>{children}</div>
      </div>
    </div>
  </ShowcaseSurface>
);
