import { FormProvider, useForm } from "react-hook-form";
import { useLocation } from "react-router";

import { UnfoldedEditor } from "@/components/agent/UnfoldedEditor";
import { EditorFooter } from "@/components/agent/EditorFooter";
import { ComputersSection } from "@/components/autoplay/ComputersSection";
import { ConditionsSection } from "@/components/autoplay/ConditionsSection";
import { LastSweepLine } from "@/components/autoplay/LastSweepLine";
import { LimitsSection } from "@/components/autoplay/LimitsSection";
import { MemoriesSection } from "@/components/autoplay/MemoriesSection";
import { MomentSection } from "@/components/autoplay/MomentSection";
import { PrioritySection } from "@/components/autoplay/PrioritySection";
import { RunOnSection } from "@/components/autoplay/RunOnSection";
import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useLatestCallback } from "@/hooks/useLatestCallback";
import { useLeaveGuard } from "@/hooks/useLeaveGuard";
import { useCreateAutoPlay, useDeleteAutoPlay, useUpdateAutoPlay } from "@/hooks/PlayHooks";
import { newDraft, subjectOf, toDraft, toSaveRequest, type AutoPlay, type AutoPlayDraft } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

const DISCARD_MESSAGE = "Your changes to this auto play will be lost.";

interface AutoPlayEditorProps {
  play: Play;
  // null composes a new one.
  autoPlay: AutoPlay | null;
  onClose: () => void;
}

// An auto play's flat editor, unfolded under its row; Save and Cancel act on this auto play.
export const AutoPlayEditor = ({ play, autoPlay, onClose }: AutoPlayEditorProps) => {
  const subject = subjectOf(play);
  const can = useAreaAccess();
  const canWrite = can?.("editAutoPlays") ?? false;
  const form = useForm<AutoPlayDraft>({
    defaultValues: autoPlay ? toDraft(autoPlay, subject) : newDraft(subject),
    disabled: !canWrite,
  });
  const { isDirty } = form.formState;
  const { pathname } = useLocation();
  const { open: confirm } = useConfirmationDialog();
  const create = useCreateAutoPlay(play.workspace_id, play.id);
  const update = useUpdateAutoPlay(play.workspace_id, play.id);
  const remove = useDeleteAutoPlay(play.workspace_id, play.id);

  useLeaveGuard(isDirty, pathname, () => undefined, DISCARD_MESSAGE);

  const back = useLatestCallback(async () => {
    if (!isDirty || (await confirm({ title: "Discard changes?", message: DISCARD_MESSAGE, confirmLabel: "Discard changes" }))) onClose();
  });

  const save = useLatestCallback(async () => {
    const input = toSaveRequest(form.getValues());
    try {
      if (autoPlay) await update.mutateAsync({ id: autoPlay.id, input: { ...input, enabled: autoPlay.enabled } });
      if (!autoPlay) await create.mutateAsync(input);
      onClose();
    } catch {
      // The hook toasts why the server refused; the editor stays open with the edits.
    }
  });

  const destroy = useLatestCallback(async () => {
    if (!autoPlay) return;
    const ok = await confirm({
      title: "Delete this auto play?",
      message: "The play stops starting itself at this moment. Runs it already started keep their trails.",
      confirmLabel: "Delete auto play",
    });
    if (ok) remove.mutate(autoPlay.id, { onSuccess: onClose });
  });

  return (
    <UnfoldedEditor label={autoPlay ? "Edit auto play" : "New auto play"}>
      <FormProvider {...form}>
        <MomentSection />
        {autoPlay?.moment === "schedule" && (
          <div className="pb-3 @xl:pl-[8.5rem]">
            <LastSweepLine playId={autoPlay.play_id} autoPlayId={autoPlay.id} />
          </div>
        )}
        <ConditionsSection />
        <RunOnSection />
        <ComputersSection />
        <PrioritySection />
        <LimitsSection />
        <MemoriesSection />
        <EditorFooter>
          {autoPlay && (
            <ActionButton permission="autoplays:delete" verb="delete auto plays" type="button" variant="ghost" liveClassName="text-destructive hover:text-destructive" onClick={() => void destroy()}>
              Delete
            </ActionButton>
          )}
          <span className="flex-1" />
          <Button type="button" variant="outline" onClick={() => void back()}>
            {canWrite ? "Cancel" : "Close"}
          </Button>
          {canWrite && (
            <Button type="button" loading={create.isPending || update.isPending} onClick={() => void save()}>
              Save
            </Button>
          )}
        </EditorFooter>
      </FormProvider>
    </UnfoldedEditor>
  );
};
