import { PlacementRow } from "@/components/agent/PlacementRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import { useProjectPlacement } from "@/hooks/PairingProjectHooks";
import { useFetchWorkspacePlays } from "@/hooks/PlayHooks";
import type { WorkspacePlay } from "@/models/Play";
import { sharedComputerName } from "@/models/SharedComputer";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface ProjectPlaysPlacementProps {
  projectId: string;
}

// The plays that run in this project, grouped by the labels they need, so plays sharing needs share one answer.
const byNeeds = (plays: WorkspacePlay[], projectId: string): Map<string, WorkspacePlay[]> => {
  const groups = new Map<string, WorkspacePlay[]>();
  for (const play of plays) {
    if (!play.enabled || play.excluded_project_ids.includes(projectId)) continue;
    const key = [...(play.needs ?? [])].sort().join(",");
    groups.set(key, [...(groups.get(key) ?? []), play]);
  }
  return groups;
};

// Where each of this project's plays would land for you now, and why the computers before it can't take it (spec decision 7).
export const ProjectPlaysPlacement = ({ projectId }: ProjectPlaysPlacementProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays } = useFetchWorkspacePlays(workspaceId);
  const groups = byNeeds(plays ?? [], projectId);
  const keys = [...groups.keys()].sort();
  const { data: answers, error, isPending } = useProjectPlacement(projectId, keys.map((key) => (key === "" ? [] : key.split(","))));
  const { data: own } = useListComputers();
  const { data: shared } = useListSharedComputers();
  const nameOf = (id: string) => {
    const computer = [...(own ?? []), ...(shared ?? [])].find((c) => c.id === id);
    return computer ? sharedComputerName(computer) : "A computer you can't see";
  };

  return (
    keys.length > 0 && (
      <section aria-labelledby="plays-land" className="space-y-2">
        <Microheader id="plays-land">Where its plays land now</Microheader>
        {isPending && <LoadingDisplay className="justify-start" />}
        {error && <ErrorDisplay error={error} />}
        {answers && (
          <ul className="divide-y divide-border rounded-md border border-border">
            {answers.map((answer, i) => (
              <PlacementRow key={keys[i]} answer={answer} plays={groups.get(keys[i] ?? "") ?? []} nameOf={nameOf} />
            ))}
          </ul>
        )}
      </section>
    )
  );
};
