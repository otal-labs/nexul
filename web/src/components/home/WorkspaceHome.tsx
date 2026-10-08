import { Link } from "react-router";

import { Container } from "@/components/Container";
import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { PlayTrailPreview } from "@/components/play/PlayTrailPreview";
import { displayTitleClass } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

// The workspace's own front page sits in the app's panel like any other page, so it gets no live field.
export const WorkspaceHome = () => {
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  const canRunPlays = useHasPermission("plays:run");
  const canReadPlays = useHasPermission("plays:read");
  // A member sees the agent side only with a way to use it.
  const showsAgent = canRunPlays || canReadPlays;

  return (
    <Container className="flex min-h-full flex-col items-center justify-center py-16 text-center">
      <Logo className="size-12 rounded-lg" />
      <p className={cn(microheaderClass, "mt-6")}>{showsAgent ? "MCP-first deployment console" : "Project workspace"}</p>
      <h1 className={cn(displayTitleClass, "mt-5 max-w-3xl text-5xl")}>
        {showsAgent ? "One button. The trail shows every step the agent took." : "Docs, tickets, chat, and deploys in one place."}
      </h1>
      <p className="mt-5 max-w-xl text-lg text-pretty text-muted-foreground">
        Nexul runs the whole loop: project management, docs as the source of truth, CI/CD runners, and deploys to
        servers you own.
        {showsAgent && " An MCP server puts every one of those tools in reach of a play, not just the browser."}
      </p>
      <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
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
      </div>
      {showsAgent && (
        <div className="mt-14 flex w-full justify-center">
          <PlayTrailPreview />
        </div>
      )}
    </Container>
  );
};
