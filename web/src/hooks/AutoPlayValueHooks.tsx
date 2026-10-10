import { useQueries } from "@tanstack/react-query";

import { personLabel } from "@nexul/client-core/person";

import { useFetchCategories } from "@/hooks/CategoryHooks";
import { docFoldersQuery } from "@/hooks/DocFolderHooks";
import { useFetchWorkspacePeople } from "@/hooks/PeopleHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { projectStatusesQuery } from "@/hooks/StatusHooks";
import { useFetchAllLabels } from "@/hooks/TicketHooks";
import { projectTicketTypesQuery } from "@/hooks/TicketTypeHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { FIELD_LABELS, ID_FIELDS, type AutoPlayField } from "@/models/AutoPlay";
import { PLAY_STAGE_LABELS, PLAY_STAGES } from "@/models/Play";
import type { Project } from "@/models/Project";

export interface ValueOption {
  value: string;
  label: string;
}

export interface FieldValues {
  options: ValueOption[];
  // Names a stored value; one the viewer can't see reads as "Unknown project" and stays in the auto play.
  label: (value: string) => string;
}

const byName = (names: string[]): ValueOption[] =>
  [...new Set(names)].sort((a, b) => a.localeCompare(b)).map((name) => ({ value: name, label: name }));

// A per-project value names its project once there are several, since two projects may share a column name.
const perProject = <T extends { id: string; name: string }>(projects: Project[], lists: (T[] | undefined)[]): ValueOption[] =>
  projects.flatMap((project, i) =>
    (lists[i] ?? []).map((item) => ({ value: item.id, label: projects.length > 1 ? `${item.name} · ${project.name}` : item.name })),
  );

// What a condition on one field may compare against, drawn from the workspace's real projects, columns, types and people.
export const useAutoPlayValues = (field: AutoPlayField): FieldValues => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects, isPending: projectsPending } = useFetchProjects();
  const { data: people } = useFetchWorkspacePeople(field === "developer" || field === "tester" ? workspaceId : undefined);
  const { data: categories } = useFetchCategories();
  const { data: labels } = useFetchAllLabels();
  const list = projects ?? [];
  const statuses = useQueries({
    queries: list.map((p) => ({ ...projectStatusesQuery(p.id), enabled: field === "status" })),
  });
  const types = useQueries({
    queries: list.map((p) => ({ ...projectTicketTypesQuery(p.id), enabled: field === "type" })),
  });
  const folders = useQueries({ queries: list.map((p) => ({ ...docFoldersQuery(p.id), enabled: field === "folder" })) });

  const options = ((): ValueOption[] => {
    if (field === "project") return list.map((p) => ({ value: p.id, label: p.name }));
    if (field === "stage") return PLAY_STAGES.map((stage) => ({ value: stage, label: PLAY_STAGE_LABELS[stage] }));
    if (field === "status") return perProject(list, statuses.map((q) => q.data));
    if (field === "folder") return perProject(list, folders.map((q) => q.data));
    if (field === "type") return byName(types.flatMap((q) => q.data ?? []).map((t) => t.name));
    if (field === "category") {
      const ids = new Set(list.map((p) => p.id));
      return byName((categories ?? []).filter((c) => ids.has(c.project_id)).map((c) => c.name));
    }
    if (field === "label") return byName(labels ?? []);
    if (field === "developer" || field === "tester") return (people ?? []).map((p) => ({ value: p.user_id, label: personLabel(p) }));
    return [];
  })();

  const pending =
    projectsPending ||
    (field === "status" && statuses.some((q) => q.isPending)) ||
    (field === "folder" && folders.some((q) => q.isPending)) ||
    ((field === "developer" || field === "tester") && !people);

  const label = (value: string) => {
    const known = options.find((option) => option.value === value);
    if (known) return known.label;
    if (!ID_FIELDS.has(field)) return value;
    if (pending) return "…";
    return `Unknown ${FIELD_LABELS[field].toLowerCase()}`;
  };

  return { options, label };
};
