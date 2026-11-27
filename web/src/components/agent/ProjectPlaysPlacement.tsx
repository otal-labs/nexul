import { PlacementList, type PlacedPlay } from "@/components/agent/PlacementList";
import { PlacementSummary } from "@/components/agent/PlacementSummary";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useListSharedComputers } from "@/hooks/ComputerHooks";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchProjectLinks, useProjectPlacement } from "@/hooks/PairingProjectHooks";
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
  const { data: links } = useFetchProjectLinks();
  const nameOf = (id: string) => {
    const computer = [...(own ?? []), ...(shared ?? [])].find((c) => c.id === id);
    return computer ? sharedComputerName(computer) : "A computer you can't see";
  };

  const placed: PlacedPlay[] = (answers ?? []).flatMap((answer, i) => (groups.get(keys[i] ?? "") ?? []).map((play) => ({ play, answer })));
  const first = placed[0]?.answer.computer_id;
  const together = !!first && placed.every((p) => p.answer.computer_id === first);
  // Computers of yours that the project's own list leaves out, which no run here tries.
  const listed = links?.find((l) => l.project_id === projectId)?.computers ?? [];
  const passedIds = new Set(placed.flatMap((p) => p.answer.passed_over.map((o) => o.computer_id)));
  const outside = listed.length === 0 ? [] : (own ?? []).filter((c) => !c.unfinished && c.id !== first && !listed.includes(c.id) && !passedIds.has(c.id)).map((c) => c.name);

  return (
    keys.length > 0 && (
      <section aria-labelledby="plays-land" className="space-y-2 py-3">
        <h3 id="plays-land" className="text-xs font-medium text-muted-foreground">
          Where its plays land now
        </h3>
        {isPending && <LoadingDisplay className="justify-start" />}
        {error && <ErrorDisplay error={error} />}
        {answers && together && <PlacementSummary count={placed.length} computer={nameOf(first)} passed={placed.flatMap((p) => p.answer.passed_over)} outside={outside} />}
        {answers && !together && <PlacementList placed={placed} nameOf={nameOf} />}
      </section>
    )
  );
};
