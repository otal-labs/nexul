import { ArrowLeftIcon } from "lucide-react";
import { FormProvider, useForm } from "react-hook-form";
import { useLocation } from "react-router";

import { ConditionsSection } from "@/components/autoplay/ConditionsSection";
import { LimitsSection } from "@/components/autoplay/LimitsSection";
import { MemoriesSection } from "@/components/autoplay/MemoriesSection";
import { MomentSection } from "@/components/autoplay/MomentSection";
import { PrioritySection } from "@/components/autoplay/PrioritySection";
import { RunOnSection } from "@/components/autoplay/RunOnSection";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useLatestCallback } from "@/hooks/useLatestCallback";
import { useLeaveGuard } from "@/hooks/useLeaveGuard";
import { useCreateAutoPlay, useUpdateAutoPlay } from "@/hooks/PlayHooks";
import { newDraft, subjectOf, toDraft, toSaveRequest, type AutoPlay, type AutoPlayDraft } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

const DISCARD_MESSAGE = "Your changes to this auto play will be lost.";

interface AutoPlayComposerProps {
  play: Play;
  // null composes a new one.
  autoPlay: AutoPlay | null;
  onClose: () => void;
}

// Takes the list's place on the play's Auto plays tab; Save and Cancel act on this auto play.
export const AutoPlayComposer = ({ play, autoPlay, onClose }: AutoPlayComposerProps) => {
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

  // Leaving the tab (another tab, another play) with edits asks first; closing the composer itself goes back to the list.
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
      // The hook toasts why the server refused; the composer stays open with the edits.
    }
  });

  return (
    <FormProvider {...form}>
      <div className="settle-in space-y-5">
        <Button type="button" variant="ghost" size="sm" className="-ml-2.5 text-muted-foreground" onClick={() => void back()}>
          <ArrowLeftIcon />
          Auto plays
        </Button>
        <MomentSection />
        <ConditionsSection />
        <PrioritySection />
        <LimitsSection />
        <RunOnSection />
        <MemoriesSection />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={() => void back()}>
            {canWrite ? "Cancel" : "Close"}
          </Button>
          {canWrite && (
            <Button type="button" loading={create.isPending || update.isPending} onClick={() => void save()}>
              Save
            </Button>
          )}
        </div>
      </div>
    </FormProvider>
  );
};
