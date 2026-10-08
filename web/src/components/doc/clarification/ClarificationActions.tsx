import { PlayButton } from "@/components/play/PlayButton";
import { Button } from "@/components/ui/button";
import { useCloseClarification, useFetchDoc } from "@/hooks/DocHooks";
import { useDocBuiltinPlays } from "@/hooks/PlayHooks";
import { answersChangedSinceWritten, clarificationPhase, type Clarification } from "@/models/Clarification";

interface ClarificationActionsProps {
  clarification: Clarification;
  docId: string;
}

// What someone who may close the clarification does next: start a round, close after no gaps, then To tickets via AI.
export const ClarificationActions = ({ clarification, docId }: ClarificationActionsProps) => {
  const { data: doc } = useFetchDoc(docId);
  const projectId = doc?.project_id ?? "";
  const { clarify, toTickets } = useDocBuiltinPlays(projectId);
  const close = useCloseClarification(docId);
  const phase = clarificationPhase(clarification);
  const changed = answersChangedSinceWritten(clarification) > 0;
  const clarifyShown = phase !== "noGaps" || changed;
  const clarifyPrimary = phase === "none" || phase === "answered" || changed;
  return (
    <span className="flex min-w-0 flex-wrap items-start gap-2">
      {clarify && clarifyShown && (
        <PlayButton play={clarify} projectId={projectId} targetType="doc" targetId={docId} variant={clarifyPrimary ? "default" : "outline"} />
      )}
      {phase === "noGaps" && (
        <Button size="sm" loading={close.isPending} onClick={() => close.mutate()}>
          Close
        </Button>
      )}
      {phase === "closed" && toTickets && (
        <PlayButton play={toTickets} projectId={projectId} targetType="doc" targetId={docId} variant="default" />
      )}
    </span>
  );
};
