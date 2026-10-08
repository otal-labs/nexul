import { Link } from "react-router";

import { Container } from "@/components/Container";
import { Logo } from "@/components/Logo";
import { PlayTrailPreview } from "@/components/play/PlayTrailPreview";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useSessionStore } from "@/stores/sessionStore";

export const HomePage = () => {
  const isLoggedIn = useSessionStore((s) => s.isLoggedIn);
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  const canRunPlays = useHasPermission("plays:run");
  const canReadPlays = useHasPermission("plays:read");
  // Visitors still reading the pitch see the agent side; a signed-in member sees it only with a way to use it.
  const showsAgent = !isLoggedIn || canRunPlays || canReadPlays;

  return (
    <div className="blueprint-bg min-h-screen">
      <Container className="flex min-h-[calc(100vh-3.5rem)] flex-col items-center justify-center py-16 text-center">
        <Logo className="size-12 rounded-xl" />
        <p className="mt-6 font-mono text-[11px] font-medium tracking-[0.24em] text-primary/90 uppercase">
          {showsAgent ? "MCP-first deployment console" : "Project workspace"}
        </p>
        <h1 className="mt-5 max-w-3xl text-4xl leading-[1.08] font-semibold tracking-tight sm:text-6xl">
          {showsAgent ? "One button. The trail shows every step the agent took." : "Docs, tickets, chat, and deploys in one place."}
        </h1>
        <p className="mt-5 max-w-xl text-base text-muted-foreground sm:text-lg">
          Nexul runs the whole loop: project management, docs as the source
          of truth, CI/CD runners, and deploys to servers you own.
          {showsAgent && " An MCP server puts every one of those tools in reach of a play, not just the browser."}
        </p>
        <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
          {isLoggedIn ? (
            <>
              {can?.("tickets") && (
                <Button asChild size="lg">
                  <Link to={wsPath("/board")}>Open the board</Link>
                </Button>
              )}
              {can?.("topology") && (
                <Button asChild variant="outline" size="lg">
                  <Link to={wsPath("/topology")}>View topology</Link>
                </Button>
              )}
            </>
          ) : (
            <>
              <Button asChild size="lg">
                <Link to="/login">Sign in</Link>
              </Button>
              <Button asChild variant="outline" size="lg">
                <a href="https://nexul.io/docs/guide/install/" target="_blank" rel="noreferrer">
                  Self-host your own
                </a>
              </Button>
            </>
          )}
        </div>
        <div className="mt-14 flex w-full justify-center">
          {showsAgent && <PlayTrailPreview />}
        </div>
      </Container>
    </div>
  );
};
