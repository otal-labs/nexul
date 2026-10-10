import { ArrowLeftIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";

import { ConditionsSection } from "@/components/autoplay/ConditionsSection";
import { LimitsSection } from "@/components/autoplay/LimitsSection";
import { MomentSection } from "@/components/autoplay/MomentSection";
import { PrioritySection } from "@/components/autoplay/PrioritySection";
import { RunOnSection } from "@/components/autoplay/RunOnSection";
import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useLatestCallback } from "@/hooks/useLatestCallback";
import { useCreateAutoPlay, useUpdateAutoPlay } from "@/hooks/PlayHooks";
import { newDraft, toDraft, toSaveRequest, type AutoPlay, type AutoPlayDraft, type AutoPlaySubject } from "@/models/AutoPlay";
import type { Play } from "@/models/Play";

interface AutoPlayComposerProps {
  play: Play;
  // null composes a new one.
  autoPlay: AutoPlay | null;
  onClose: () => void;
}

// Takes the list's place in the play's dialog; while open, the dialog's Save and Cancel act on this auto play.
export const AutoPlayComposer = ({ play, autoPlay, onClose }: AutoPlayComposerProps) => {
  const subject: AutoPlaySubject = play.type === "doc" ? "doc" : "ticket";
  const can = useAreaAccess();
  const canWrite = can?.("editAutoPlays") ?? false;
  const form = useForm<AutoPlayDraft>({
    defaultValues: autoPlay ? toDraft(autoPlay, subject) : newDraft(subject),
    disabled: !canWrite,
  });
  const { isDirty } = form.formState;
  const { intercept } = useFormDialogContext();
  const { open: confirm } = useConfirmationDialog();
  const create = useCreateAutoPlay(play.workspace_id, play.id);
  const update = useUpdateAutoPlay(play.workspace_id, play.id);

  const back = useLatestCallback(async () => {
    if (isDirty) {
      const discard = await confirm({
        title: "Discard changes?",
        message: "Your changes to this auto play will be lost.",
        confirmLabel: "Discard changes",
      });
      if (!discard) return;
    }
    onClose();
  });

  const save = useLatestCallback(async () => {
    if (!canWrite) {
      onClose();
      return;
    }
    const input = toSaveRequest(form.getValues());
    try {
      if (autoPlay) await update.mutateAsync({ id: autoPlay.id, input: { ...input, enabled: autoPlay.enabled } });
      if (!autoPlay) await create.mutateAsync(input);
      onClose();
    } catch {
      // The hook toasts why the server refused; the composer stays open with the edits.
    }
  });

  useEffect(() => {
    intercept({ submit: save, cancel: () => void back() });
    return () => intercept(null);
  }, [intercept, save, back]);

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
      </div>
    </FormProvider>
  );
};
