import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PlayRunForm } from "@/components/play/PlayRunForm";
import { UnansweredQuestionsSignal } from "@/components/play/UnansweredQuestionsSignal";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useHarnessReadiness } from "@/hooks/PairingHooks";
import { useFetchProjectLinks } from "@/hooks/PairingProjectHooks";
import { useFetchLatestChoices } from "@/hooks/TrailHooks";
import type { Play, PlayType } from "@/models/Play";

interface PlayRunDialogProps {
  play: Play;
  projectId: string;
  targetType: PlayType;
  targetId: string;
  open: boolean;
  onClose: () => void;
  // Seeds the run's own instructions and opens Where to run, for a run started again after it asked where.
  instructions?: string;
  askWhere?: boolean;
}

interface PlayRunDialogBodyProps {
  play: Play;
  projectId: string;
  targetType: PlayType;
  targetId: string;
  instructions: string;
  askWhere: boolean;
  onDone: () => void;
}

// Loads what the form is seeded from; the form mounts once everything is here so its initial state needs no effect.
const PlayRunDialogBody = ({ play, projectId, targetType, targetId, instructions, askWhere, onDone }: PlayRunDialogBodyProps) => {
  const memories = useFetchMemoriesByProject(projectId);
  const choices = useFetchLatestChoices(play.id, projectId);
  // Shares the resolve query's cache with the play button that gated this dialog open, so this costs no extra call.
  const readiness = useHarnessReadiness(projectId);
  const links = useFetchProjectLinks();
  const isPending = memories.isPending || choices.isPending || links.isPending || readiness === undefined;
  const error = memories.error ?? choices.error ?? links.error;
  const ready = readiness?.state === "ready" ? readiness : undefined;
  const resolvedHarness = {
    computer_id: ready?.computerId ?? "",
    provider: ready?.provider ?? "",
    model: ready?.model ?? "",
    model_options: ready?.modelOptions ?? [],
  };
  const link = links.data?.find((l) => l.project_id === projectId);
  // Unlinked, readiness resolved the person's defaults: offered as the suggestion, never run on unasked (ADR 0145).
  const where = link?.computer_id
    ? { computer_id: link.computer_id, harness_project_id: link.harness_project_id ?? "" }
    : { computer_id: resolvedHarness.computer_id, harness_project_id: ready?.harnessProjectId ?? "" };

  return (
    <div className="space-y-5">
      {isPending && <LoadingDisplay label="Loading choices…" />}
      {error && <ErrorDisplay error={error} title="Couldn't load the run choices." />}
      {play.builtin_key === "clarify" && targetType === "doc" && <UnansweredQuestionsSignal docId={targetId} />}
      {memories.data && choices.data && links.data && readiness && (
        <PlayRunForm
          play={play}
          targetType={targetType}
          targetId={targetId}
          memories={memories.data}
          choices={choices.data}
          resolvedHarness={resolvedHarness}
          where={where}
          linked={!!link?.computer_id}
          instructions={instructions}
          askWhere={askWhere}
          onDone={onDone}
        />
      )}
    </div>
  );
};

// The content unmounts on close, so every open re-reads the caller's latest choices.
export const PlayRunDialog = ({ play, projectId, targetType, targetId, open, onClose, instructions = "", askWhere = false }: PlayRunDialogProps) => (
  <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
    <DialogContent className="gap-5 sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{play.label}</DialogTitle>
        <DialogDescription>
          {play.description !== "" && `${play.description.replace(/\.+$/, "")}. `}Runs on your paired harness as you.
        </DialogDescription>
      </DialogHeader>
      <PlayRunDialogBody
        play={play}
        projectId={projectId}
        targetType={targetType}
        targetId={targetId}
        instructions={instructions}
        askWhere={askWhere}
        onDone={onClose}
      />
    </DialogContent>
  </Dialog>
);
