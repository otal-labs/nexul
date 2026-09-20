import { Link } from "react-router";

import { Container } from "@/components/Container";
import { Logo } from "@/components/Logo";
import { PlayTrailPreview } from "@/components/play/PlayTrailPreview";
import { Button } from "@/components/ui/button";
import { useSessionStore } from "@/stores/sessionStore";

export const HomePage = () => {
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);

  return (
    <div className="blueprint-bg min-h-screen">
      <Container className="flex min-h-[calc(100vh-3.5rem)] flex-col items-center justify-center py-16 text-center">
        <Logo className="size-12 rounded-xl" />
        <p className="mt-6 font-mono text-[11px] font-medium tracking-[0.24em] text-primary/90 uppercase">
          MCP-first deployment console
        </p>
        <h1 className="mt-5 max-w-3xl text-4xl leading-[1.08] font-semibold tracking-tight sm:text-6xl">
          One button. The trail shows every step the agent took.
        </h1>
        <p className="mt-5 max-w-xl text-base text-muted-foreground sm:text-lg">
          Nexul runs the whole loop: project management, docs as the source
          of truth, CI/CD runners, and deploys to servers you own. An MCP
          server puts every one of those tools in reach of a play, not just
          the browser.
        </p>
        <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
          {isLoggedIn ? (
            <>
              <Button asChild size="lg">
                <Link to="/board">Open the board</Link>
              </Button>
              <Button asChild variant="outline" size="lg">
                <Link to="/topology">View topology</Link>
              </Button>
            </>
          ) : (
            <>
              <Button asChild size="lg">
                <Link to="/login">Sign in with GitHub</Link>
              </Button>
              <Button asChild variant="outline" size="lg">
                <Link to="/login">Self-host your own</Link>
              </Button>
            </>
          )}
        </div>
        <div className="mt-14 flex w-full justify-center">
          <PlayTrailPreview />
        </div>
      </Container>
    </div>
  );
};
