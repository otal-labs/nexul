import { useParams } from "react-router";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { ListPaneGroup } from "@/components/listpane/ListPaneGroup";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { groupPlaysByType } from "@/components/play/playGroups";
import { PlayListRow } from "@/components/play/PlayListRow";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import { PLAY_TYPE_LABELS, type WorkspacePlay } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface AgentPlayGroupsProps {
  query: string;
}

const matches = (play: WorkspacePlay, query: string) => [play.label, play.description].some((text) => text.toLowerCase().includes(query));

// The workspace's plays, one group per type; the list refreshes from the plays the socket pushes.
export const AgentPlayGroups = ({ query }: AgentPlayGroupsProps) => {
  const { kind, id } = useParams();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays, error, isPending } = useFetchWorkspacePlays(workspaceId);
  const groups = plays && groupPlaysByType(query === "" ? plays : plays.filter((play) => matches(play, query)));
  const selectedId = kind === "plays" ? id : undefined;

  return (
    <div>
      {isPending && <LoadingDisplay label="Loading plays…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load plays." />}
      {plays && plays.length === 0 && (
        <ListPaneGroup label="Plays" count={0}>
          <li>
            <EmptyRow className="px-3 py-2 text-xs">No plays yet</EmptyRow>
          </li>
        </ListPaneGroup>
      )}
      {groups?.map((group) => (
        <ListPaneGroup key={group.type} label={`${PLAY_TYPE_LABELS[group.type]} plays`} count={group.plays.length}>
          {group.plays.map((play) => (
            <PlayListRow key={play.id} play={play} selected={play.id === selectedId} />
          ))}
        </ListPaneGroup>
      ))}
    </div>
  );
};
