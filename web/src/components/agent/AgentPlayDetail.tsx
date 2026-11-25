import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PlayDetail } from "@/components/play/PlayDetail";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface AgentPlayDetailProps {
  playId: string;
}

// Read from the plays the list already holds, so opening one fetches nothing more.
export const AgentPlayDetail = ({ playId }: AgentPlayDetailProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays, error, isPending } = useFetchWorkspacePlays(workspaceId);
  const play = plays?.find((p) => p.id === playId);

  return (
    <div>
      {isPending && <LoadingDisplay label="Loading plays…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load plays." />}
      {play && <PlayDetail play={play} />}
      {plays && !play && <NoDataDisplay message="That play isn't here" />}
    </div>
  );
};
