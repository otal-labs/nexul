import { Link } from "react-router";

import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { PlayTrailPreview } from "@/components/play/PlayTrailPreview";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export const SignedOutHome = () => (
  <div className="mx-auto flex min-h-full w-full max-w-6xl flex-col px-8 lg:px-12">
    <header className="flex items-center gap-2.5 py-6 text-[15px] font-semibold tracking-tight">
      <Logo className="size-7 rounded-md" />
      Nexul
    </header>
    <section className="flex flex-1 flex-col justify-center py-14">
      <p className={microheaderClass}>Open source, self-hosted</p>
      <h1 className={cn(displayTitleClass, "mt-5 max-w-4xl text-[clamp(2.75rem,6.4vw,4.75rem)]")}>
        Docs, tickets, chat, and deploys in one place.
      </h1>
      <div className="mt-12 grid items-end gap-12 lg:grid-cols-[minmax(0,26rem)_minmax(0,1fr)] lg:gap-16">
        <div>
          <p className="text-lg text-pretty text-muted-foreground">
            Runners build and deploy to servers you own. Agents use the same tools over MCP, and each play run
            leaves a trail of every step.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Button asChild size="lg">
              <Link to="/login">Sign in</Link>
            </Button>
            <Button asChild variant="outline" size="lg">
              <a href="https://nexul.io/docs/guide/install/" target="_blank" rel="noreferrer">
                Self-host your own
              </a>
            </Button>
          </div>
        </div>
        <PlayTrailPreview />
      </div>
    </section>
    <footer className="flex items-center justify-between border-t border-border py-6 font-mono text-xs text-muted-foreground">
      <span>self-hosted · sqlite · no cloud required</span>
      <a href="https://nexul.io" target="_blank" rel="noreferrer" className="hover:text-foreground">
        nexul.io
      </a>
    </footer>
  </div>
);
