import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { PlayButton } from "@/components/play/PlayButton";
import { Button } from "@/components/ui/button";
import { useCloseClarification, useFetchDoc } from "@/hooks/DocHooks";
import { useDocBuiltinPlays } from "@/hooks/PlayHooks";
import { useFetchTicketsByDoc } from "@/hooks/TicketHooks";
import { answersChangedSinceWritten, clarificationPhase, type Clarification } from "@nexul/client-core/clarification";

interface ClarificationActionsProps {
  clarification: Clarification;
  docId: string;
}

// What someone who may close the clarification does next: start or continue rounds, write the answers in, close, then To tickets via AI until the doc has tickets.
export const ClarificationActions = ({ clarification, docId }: ClarificationActionsProps) => {
  const { data: doc } = useFetchDoc(docId);
  const projectId = doc?.project_id ?? "";
  const { clarify, updateDoc, toTickets } = useDocBuiltinPlays(projectId);
  const close = useCloseClarification(docId);
  const { data: tickets } = useFetchTicketsByDoc(docId);
  const phase = clarificationPhase(clarification);
  const changed = answersChangedSinceWritten(clarification) > 0;
  const clarifyShown = phase !== "noGaps" || changed;
  const updateShown = !!updateDoc && (phase === "answered" || (phase === "noGaps" && changed));
  const clarifyPrimary = !updateShown && (phase === "none" || phase === "answered" || changed);
  const toTicketsShown = phase === "closed" && !!toTickets && tickets?.length === 0;
  const playShown = (!!clarify && clarifyShown) || updateShown || toTicketsShown;
  return (
    <span className="flex min-w-0 flex-wrap items-start gap-2">
      {updateDoc && updateShown && <PlayButton play={updateDoc} projectId={projectId} targetType="doc" targetId={docId} variant="default" />}
      {clarify && clarifyShown && (
        <PlayButton
          play={clarify}
          projectId={projectId}
          targetType="doc"
          targetId={docId}
          variant={clarifyPrimary ? "default" : "outline"}
          label={phase === "none" || phase === "closed" ? clarify.label : "Continue via AI"}
        />
      )}
      {phase === "noGaps" && (
        <Button size="sm" loading={close.isPending} onClick={() => close.mutate()}>
          Close
        </Button>
      )}
      {toTicketsShown && toTickets && (
        <PlayButton play={toTickets} projectId={projectId} targetType="doc" targetId={docId} variant="default" />
      )}
      {playShown && <HarnessReadinessNote projectId={projectId} className="basis-full font-mono" />}
    </span>
  );
};
