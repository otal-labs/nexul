import { SparklesIcon } from "lucide-react";
import { useParams } from "react-router";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListDetailLayout } from "@/components/listpane/ListDetailLayout";
import { ListDetailPlaceholder } from "@/components/listpane/ListDetailPlaceholder";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PlayDetail } from "@/components/play/PlayDetail";
import { PlaysListPane } from "@/components/play/PlaysListPane";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// The workspace's plays beside the open one; the list refreshes from the plays the socket pushes.
export const PlaysPage = () => {
  const { playId } = useParams();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const workspace = useSelectedWorkspace();
  const { data: plays, error, isPending } = useFetchWorkspacePlays(workspaceId);
  const selected = plays?.find((play) => play.id === playId);

  return (
    <div className="h-full">
      {isPending && <LoadingDisplay label="Loading plays…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load plays." />}
      {plays && (
        <ListDetailLayout
          hasSelection={!!playId}
          stableGutter
          list={<PlaysListPane plays={plays} selectedId={playId} />}
          placeholder={
            <ListDetailPlaceholder
              icon={SparklesIcon}
              title="Select a play"
              summary={plays.length === 0 ? "No plays yet" : `${plays.length} ${plays.length === 1 ? "play" : "plays"} in ${workspace?.name ?? "this workspace"}`}
            />
          }
          detail={
            <>
              {selected && <PlayDetail key={selected.id} play={selected} />}
              {!selected && <NoDataDisplay message="That play isn't here" />}
            </>
          }
        />
      )}
    </div>
  );
};
