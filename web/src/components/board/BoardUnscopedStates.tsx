import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PageHeader } from "@/components/PageHeader";
import { NoProjectsState } from "@/components/project/NoProjectsState";

interface BoardUnscopedStatesProps {
  isLoading: boolean;
  error: unknown;
  hasProjects: boolean;
}

// The unscoped /board route while it resolves a redirect; zero projects is the one dead end it can't recover from.
export const BoardUnscopedStates = ({ isLoading, error, hasProjects }: BoardUnscopedStatesProps) => (
  <>
    <PageHeader title="Board" />
    {isLoading && <LoadingDisplay label="Loading board…" />}
    {!isLoading && error && <ErrorDisplay error={error} title="Couldn't load the board." />}
    {!isLoading && !error && !hasProjects && (
      <NoProjectsState message="Create one to start its board." />
    )}
    {!isLoading && !error && hasProjects && <LoadingDisplay label="Loading board…" />}
  </>
);
