import { Controller } from "react-hook-form";

import { PermissionCheckRow } from "@/components/access/PermissionCheckRow";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useCreatePlay, useUpdatePlay } from "@/hooks/PlayHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { PLAY_STAGE_LABELS, PLAY_STAGES, type Play, type SavePlayFormData } from "@/models/Play";

const TYPE_OPTIONS = [
  { value: "ticket", label: "Ticket" },
  { value: "doc", label: "Doc" },
];

const STAGE_OPTIONS = PLAY_STAGES.map((stage) => ({ value: stage, label: PLAY_STAGE_LABELS[stage] }));

interface PlayFormProps {
  workspaceId: string;
  // Present when editing; type is immutable after create (ticket 20), so the field renders as text, not a select.
  editing?: Play;
}

export const PlayForm = ({ workspaceId, editing }: PlayFormProps) => {
  const { control, watch, setValue, onSubmit } = useFormDialogContext<SavePlayFormData>();
  const { data: projects } = useFetchProjects();
  const createPlay = useCreatePlay(workspaceId);
  const updatePlay = useUpdatePlay(workspaceId);
  const type = watch("type");
  const excludedProjectIds = watch("excluded_project_ids");

  onSubmit(async (input) => {
    const play = editing
      ? await updatePlay.mutateAsync({ playId: editing.id, input })
      : await createPlay.mutateAsync(input);
    return { id: play.id, ...input };
  });

  const toggleProject = (projectId: string) => {
    const next = excludedProjectIds.includes(projectId)
      ? excludedProjectIds.filter((id) => id !== projectId)
      : [...excludedProjectIds, projectId];
    setValue("excluded_project_ids", next, { shouldDirty: true });
  };

  return (
    <div className="space-y-4">
      <FormInput control={control} name="label" label="Label" placeholder="e.g. Fix with AI" autoFocus />

      {!editing && (
        <FormSelect
          control={control}
          name="type"
          label="Type"
          options={TYPE_OPTIONS}
          // A doc play carries no stage and a ticket play always needs one; switching type keeps that in sync.
          onChangeValue={(value) => setValue("show_when_stage", value === "ticket" ? "progress" : "")}
        />
      )}
      {editing && (
        <p className="text-sm text-muted-foreground">
          Type: <span className="font-medium text-foreground">{editing.type === "ticket" ? "Ticket" : "Doc"}</span>
          {" — can't change after create"}
        </p>
      )}

      {type === "ticket" && (
        <FormSelect
          control={control}
          name="show_when_stage"
          label="Show when"
          options={STAGE_OPTIONS}
          placeholder="No stage"
        />
      )}

      <div className="space-y-2">
        <label htmlFor="description" className="text-sm font-medium">
          Description
        </label>
        <Controller
          control={control}
          name="description"
          render={({ field }) => (
            <Textarea id="description" placeholder="One line shown as a tooltip" {...field} />
          )}
        />
      </div>

      <div className="space-y-2">
        <label htmlFor="instructions" className="text-sm font-medium">
          Instructions
        </label>
        <Controller
          control={control}
          name="instructions"
          render={({ field }) => (
            <Textarea id="instructions" placeholder="What the Agent should do on this run" rows={6} {...field} />
          )}
        />
      </div>

      <div className="flex items-center justify-between rounded-md border border-border p-3">
        <label htmlFor="enabled" className="text-sm font-medium">
          Enabled
        </label>
        <Controller
          control={control}
          name="enabled"
          render={({ field }) => (
            <Switch id="enabled" checked={field.value} onCheckedChange={field.onChange} />
          )}
        />
      </div>

      <div className="space-y-2">
        <span className="text-sm font-medium">Excluded projects</span>
        <div className="max-h-40 space-y-1 overflow-y-auto rounded-lg border p-2">
          {projects?.map((project) => (
            <PermissionCheckRow
              key={project.id}
              checked={excludedProjectIds.includes(project.id)}
              onCheckedChange={() => toggleProject(project.id)}
            >
              <span>{project.name}</span>
            </PermissionCheckRow>
          ))}
          {projects && projects.length === 0 && (
            <p className="px-2 py-1 text-sm text-muted-foreground">No projects yet</p>
          )}
        </div>
      </div>
    </div>
  );
};
