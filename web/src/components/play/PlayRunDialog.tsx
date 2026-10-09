import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlayRunForm } from "@/components/play/PlayRunForm";
import { UnansweredQuestionsSignal } from "@/components/play/UnansweredQuestionsSignal";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useHarnessReadiness } from "@/hooks/PairingHooks";
import { useFetchLatestChoices } from "@/hooks/TrailHooks";
import type { Play, PlayType } from "@/models/Play";

interface PlayRunDialogProps {
  play: Play;
  projectId: string;
  targetType: PlayType;
  targetId: string;
  open: boolean;
  onClose: () => void;
}

interface PlayRunDialogBodyProps {
  play: Play;
  projectId: string;
  targetType: PlayType;
  targetId: string;
  onDone: () => void;
}

// Loads what the form is seeded from; the form mounts once everything is here so its initial state needs no effect.
const PlayRunDialogBody = ({ play, projectId, targetType, targetId, onDone }: PlayRunDialogBodyProps) => {
  const memories = useFetchMemoriesByProject(projectId);
  const choices = useFetchLatestChoices(play.id, projectId);
  // Shares the resolve query's cache with the play button that gated this dialog open, so this costs no extra call.
  const readiness = useHarnessReadiness(projectId);
  const isPending = memories.isPending || choices.isPending || readiness === undefined;
  const error = memories.error ?? choices.error;
  const resolvedHarness =
    readiness?.state === "ready"
      ? { computer_id: readiness.computerId, provider: readiness.provider, model: readiness.model, model_options: readiness.modelOptions }
      : { computer_id: "", provider: "", model: "", model_options: [] };

  return (
    <div className="space-y-5">
      {isPending && <LoadingDisplay label="Loading choices…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load the run choices." />}
      {play.builtin_key === "clarify" && targetType === "doc" && <UnansweredQuestionsSignal docId={targetId} />}
      {memories.data && choices.data && readiness && (
        <PlayRunForm
          play={play}
          targetType={targetType}
          targetId={targetId}
          memories={memories.data}
          choices={choices.data}
          resolvedHarness={resolvedHarness}
          onDone={onDone}
        />
      )}
    </div>
  );
};

// The content unmounts on close, so every open re-reads the caller's latest choices.
export const PlayRunDialog = ({ play, projectId, targetType, targetId, open, onClose }: PlayRunDialogProps) => (
  <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
    <DialogContent className="gap-5 sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{play.label}</DialogTitle>
        <DialogDescription>
          {play.description !== "" && `${play.description.replace(/\.+$/, "")}. `}Runs on your paired harness as you.
        </DialogDescription>
      </DialogHeader>
      <PlayRunDialogBody play={play} projectId={projectId} targetType={targetType} targetId={targetId} onDone={onClose} />
    </DialogContent>
  </Dialog>
);
