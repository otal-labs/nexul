import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";

interface BoardUnscopedStatesProps {
  isLoading: boolean;
  error: unknown;
  hasProjects: boolean;
  onCreateProject: () => void;
}

// The unscoped /board route while it resolves a redirect; zero projects is the one dead end it can't recover from.
export const BoardUnscopedStates = ({ isLoading, error, hasProjects, onCreateProject }: BoardUnscopedStatesProps) => (
  <>
    <PageHeader title="Board" subtitle="Every ticket in its lane, traffic optional." />
    {isLoading && <LoadingDisplay label="Loading board…" />}
    {!isLoading && error && <ErrorDisplay error={error} title="Failed to load the board." />}
    {!isLoading && !error && !hasProjects && (
      <EmptyState
        title="No projects yet"
        message="Create a project to start building its board."
        action={
          <Button size="sm" onClick={onCreateProject}>
            New project
          </Button>
        }
      />
    )}
    {!isLoading && !error && hasProjects && <LoadingDisplay label="Loading board…" />}
  </>
);
