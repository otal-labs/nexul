import { useMemo } from "react";
import { useFormContext } from "react-hook-form";

import { FormCombobox, type ComboboxOption } from "@/components/FormCombobox";
import { useFetchDocs } from "@/hooks/DocHooks";
import { useFetchMemories } from "@/hooks/MemoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { SOURCE_KIND_LABEL, type AddSourceFormData } from "@/models/InterviewSource";

interface InterviewSourcePickerProps {
  kind: "doc" | "memory" | "project";
  projectId: string;
}

// The workspace's docs, memories, or other projects to point the interview at, each doc and memory with its project.
export const InterviewSourcePicker = ({ kind, projectId }: InterviewSourcePickerProps) => {
  const { control } = useFormContext<AddSourceFormData>();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: projects } = useFetchProjects();
  const { data: docs } = useFetchDocs();
  const { data: memories } = useFetchMemories(workspaceId);

  const options = useMemo((): ComboboxOption[] => {
    const names = new Map((projects ?? []).map((p) => [p.id, p.name]));
    const hint = (id: string) => names.get(id) ?? "";
    if (kind === "project") return (projects ?? []).filter((p) => p.id !== projectId).map((p) => ({ value: p.id, label: p.name }));
    if (kind === "memory") return (memories ?? []).map((m) => ({ value: m.id, label: m.title, hint: hint(m.project_id) }));
    return (docs ?? [])
      .filter((d) => names.has(d.project_id) && d.can_open && !d.archived)
      .map((d) => ({ value: d.id, label: d.title, hint: hint(d.project_id) }));
  }, [kind, projectId, projects, docs, memories]);

  const label = SOURCE_KIND_LABEL[kind];
  return <FormCombobox control={control} name="ref" label={label} placeholder={`Choose a ${label.toLowerCase()}`} options={options} />;
};
