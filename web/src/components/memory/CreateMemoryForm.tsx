import { useEffect } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { Switch } from "@/components/ui/switch";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useCreateMemory } from "@/hooks/MemoryHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { CreateMemoryFormData } from "@/models/Memory";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface CreateMemoryFormProps {
  defaultProjectId?: string;
}

// Body is edited on the memory page after creation, mirroring how CreateDocForm's dialog stays lighter than the
// full editor; unlike docs, a memory's create dialog never touches the rich-text body at all.
export const CreateMemoryForm = ({ defaultProjectId = "" }: CreateMemoryFormProps) => {
  const { control, setValue, watch, onSubmit, setLoading } = useFormDialogContext<CreateMemoryFormData>();
  const createMemory = useCreateMemory();
  const { data: projects } = useFetchProjects();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);

  const ready = projects != null;

  useEffect(() => {
    setLoading(!ready);
  }, [ready, setLoading]);

  useEffect(() => {
    if (!ready) return;
    setValue("project_id", defaultProjectId);
  }, [ready, defaultProjectId, setValue]);

  useEffect(() => {
    setValue("workspace_id", workspaceId);
  }, [workspaceId, setValue]);

  onSubmit(async (input) => {
    const memory = await createMemory.mutateAsync(input);
    return { id: memory.id, ...input };
  });

  const projectOptions = (projects ?? []).map((project) => ({ value: project.id, label: project.name }));

  return (
    <div className="space-y-4">
      {/* "Workspace" rides the placeholder slot FormSelect already has for a "" value (see its Radix comment). */}
      <FormSelect
        control={control}
        name="project_id"
        label="Project"
        options={projectOptions}
        placeholder="Workspace"
      />
      <FormInput control={control} name="title" label="Title" placeholder="Memory title" />
      <FormInput
        control={control}
        name="when_to_use"
        label="When to use"
        placeholder="use this if you are writing React code"
      />
      <label className="flex items-center gap-2 text-sm font-medium">
        <Switch
          checked={watch("always_included")}
          onCheckedChange={(checked) => setValue("always_included", checked)}
          aria-label="Always included"
        />
        Always included in every turn
      </label>
    </div>
  );
};
