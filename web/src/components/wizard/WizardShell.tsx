import { useState, type ReactNode } from "react";

import { Logo } from "@/components/Logo";
import { WizardBarContext } from "@/hooks/useWizardBar";
import { cn } from "@/lib/utils";

interface WizardShellProps {
  rail: ReactNode;
  // The way out, beside the logo; a flow that can't be left passes none.
  exit?: ReactNode;
  children: ReactNode;
}

// 40rem of reading width inside the gutters at every size.
const column = "mx-auto w-full max-w-[43rem] px-6 @4xl:max-w-[45rem] @4xl:px-10";

// Every wizard's one frame: a well-toned rail column with the logo and the way out, the step beside it fading into the action bar pinned under it on its own tracks.
export const WizardShell = ({ rail, exit, children }: WizardShellProps) => {
  const [bar, setBar] = useState<HTMLElement | null>(null);
  return (
    <div className="@container h-full">
      <div className="grid h-full grid-cols-[13rem_minmax(0,1fr)] text-pretty @4xl:grid-cols-[15rem_minmax(0,1fr)]">
        <aside className="min-h-0 overflow-y-auto bg-surface-2 px-5 py-5">
          <header className="mb-8 flex h-8 items-center justify-between">
            <Logo className="size-7" />
            {exit}
          </header>
          {rail}
        </aside>
        <div className="grid min-h-0 grid-rows-[minmax(0,1fr)_4rem]">
          <div className="mt-2 mr-2 min-h-0 overflow-y-auto [mask-image:linear-gradient(to_bottom,black_calc(100%-1.5rem),transparent)] [scrollbar-gutter:stable]">
            <div className={cn(column, "py-8 @4xl:py-12")}>
              <WizardBarContext value={bar}>{children}</WizardBarContext>
            </div>
          </div>
          <div className="border-t border-border">
            <div className="mr-2 h-full overflow-hidden [scrollbar-gutter:stable]">
              <div ref={setBar} className={cn(column, "flex h-full items-center")} />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
