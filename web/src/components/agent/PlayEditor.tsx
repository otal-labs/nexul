import { HistoryIcon } from "lucide-react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { PLAY_STAGE_LABELS, PLAY_STAGES } from "@nexul/client-core/plays";

import { ActionButton } from "@/components/action/ActionButton";
import { ComposerSection, InlineSelect } from "@/components/autoplay/ComposerControls";
import { EditorFooter } from "@/components/agent/EditorFooter";
import { PlayProjectsPicker } from "@/components/agent/PlayProjectsPicker";
import { UnfoldedEditor } from "@/components/agent/UnfoldedEditor";
import { PlayNeedsField } from "@/components/play/PlayNeedsField";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { playLabelError, useUpdatePlay } from "@/hooks/PlayHooks";
import { useActionState } from "@/hooks/useActionState";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useConfirmPlayRename } from "@/hooks/useConfirmPlayRename";
import { useEntityActions } from "@/hooks/useEntityActions";
import { usePlayMenu } from "@/hooks/usePlayMenu";
import { PLAY_REPOSITORIES, PLAY_REPOSITORY_LABELS, playToFormData, SavePlayFormSchema, type Play, type SavePlayFormData } from "@/models/Play";

const STAGE_OPTIONS = PLAY_STAGES.map((stage) => ({ value: stage, label: PLAY_STAGE_LABELS[stage] }));
const REPOSITORY_OPTIONS = PLAY_REPOSITORIES.map((repository) => ({ value: repository, label: PLAY_REPOSITORY_LABELS[repository] }));

interface PlayEditorProps {
  play: Play;
  onClose: () => void;
  onHistory: () => void;
}

// A play's own settings unfolded under its row: the same fields and rules as its create form, laid out as label rows.
export const PlayEditor = ({ play, onClose, onHistory }: PlayEditorProps) => {
  const answer = useEntityActions("play", play.id);
  const write = useActionState({ permission: "plays:write", answer: answer("plays:write"), verb: "edit plays" });
  const form = useForm<SavePlayFormData>({ resolver: zodResolver(SavePlayFormSchema), defaultValues: playToFormData(play), disabled: write.state !== "live" });
  const { control, register, setValue, setError, watch, formState } = form;
  const update = useUpdatePlay(play.workspace_id);
  const confirmRename = useConfirmPlayRename(play.workspace_id);
  const { open: confirm } = useConfirmationDialog();
  const { onDelete } = usePlayMenu(play);
  const stage = watch("show_when_stage");
  const repository = watch("repository");
  const excluded = watch("excluded_project_ids");

  const back = async () => {
    if (!formState.isDirty || (await confirm({ title: "Discard changes?", message: "Your changes to this play will be lost.", confirmLabel: "Discard changes" }))) onClose();
  };

  const save = form.handleSubmit(async (input) => {
    if (!(await confirmRename(play, input.label))) {
      setValue("label", play.label);
      return;
    }
    try {
      await update.mutateAsync({ playId: play.id, input });
      onClose();
    } catch (error) {
      const taken = playLabelError(error);
      if (taken) setError("label", { type: "server", message: taken }, { shouldFocus: true });
    }
  });

  return (
    <UnfoldedEditor label={`Edit ${play.label}`}>
      <ComposerSection title="Name">
        <Input aria-label="Name" aria-invalid={!!formState.errors.label || undefined} className="max-w-md" {...register("label")} />
        {formState.errors.label && <p className="text-xs text-destructive">{formState.errors.label.message}</p>}
      </ComposerSection>
      {play.type === "ticket" && (
        <ComposerSection title="Button shows in">
          <InlineSelect label="Button shows in" value={stage} options={STAGE_OPTIONS} onChange={(value) => setValue("show_when_stage", value, { shouldDirty: true })} disabled={formState.disabled} />
          <p className="text-xs text-muted-foreground">The stage where its button appears on a ticket</p>
        </ComposerSection>
      )}
      <ComposerSection title="Description">
        <Textarea aria-label="Description" rows={2} placeholder="One line shown as a tooltip" {...register("description")} />
      </ComposerSection>
      <ComposerSection title="Instructions">
        <Textarea aria-label="Instructions" rows={8} placeholder="What the Agent should do on this run" className="technical max-h-52 overflow-y-auto" {...register("instructions")} />
      </ComposerSection>
      <ComposerSection title="Works in">
        <InlineSelect label="Works in" value={repository} options={REPOSITORY_OPTIONS} onChange={(value) => setValue("repository", value as Play["repository"], { shouldDirty: true })} disabled={formState.disabled} />
      </ComposerSection>
      <ComposerSection title="Projects">
        <PlayProjectsPicker excluded={excluded} onChange={(next) => setValue("excluded_project_ids", next, { shouldDirty: true })} disabled={!!formState.disabled} />
        <p className="text-xs text-muted-foreground">Leave a project out and its button disappears there</p>
      </ComposerSection>
      <ComposerSection title="Needs">
        <PlayNeedsField control={control} bare />
      </ComposerSection>
      <EditorFooter>
        {onDelete && (
          <ActionButton permission="plays:delete" verb="delete plays" type="button" variant="ghost" liveClassName="text-destructive hover:text-destructive" onClick={onDelete}>
            Delete
          </ActionButton>
        )}
        <ActionButton permission="plays:read" verb="read play history" icon={HistoryIcon} type="button" variant="outline" onClick={onHistory}>
          History
        </ActionButton>
        <span className="flex-1" />
        <Button type="button" variant="outline" onClick={() => void back()}>
          Cancel
        </Button>
        {write.state === "live" && (
          <Button type="button" loading={update.isPending} onClick={() => void save()}>
            Save
          </Button>
        )}
      </EditorFooter>
    </UnfoldedEditor>
  );
};
