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
      <p className={cn(microheaderClass, "mt-6")}>Self-hosted workspace</p>
      <h1 className={cn(displayTitleClass, "mt-5 max-w-3xl text-5xl")}>
        Docs, tickets, chat, and deploys in one place.
      </h1>
      <p className="mt-5 max-w-xl text-lg text-pretty text-muted-foreground">
        Runners build and deploy to servers you own.
        {showsAgent && " Agents use the same tools over MCP, and each play run leaves a trail of every step."}
      </p>
      <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
        {can?.("tickets") && (
          <Button asChild size="lg">
            <Link to={wsPath("/board")}>Open board</Link>
          </Button>
        )}
        {can?.("topology") && (
          <Button asChild variant="outline" size="lg">
            <Link to={wsPath("/topology")}>Open topology</Link>
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
