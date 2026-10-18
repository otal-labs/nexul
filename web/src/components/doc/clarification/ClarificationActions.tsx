import { HarnessReadinessNote } from "@/components/play/HarnessReadinessNote";
import { PlayButton } from "@/components/play/PlayButton";
import { Button } from "@/components/ui/button";
import { useCloseClarification, useFetchDoc } from "@/hooks/DocHooks";
import { useDocBuiltinPlays } from "@/hooks/PlayHooks";
import { answersChangedSinceWritten, clarificationPhase, type Clarification } from "@nexul/client-core/clarification";

interface ClarificationActionsProps {
  clarification: Clarification;
  docId: string;
}

// What someone who may close the clarification does next: start a round, write the answers in once none waits, close, then To tickets via AI.
export const ClarificationActions = ({ clarification, docId }: ClarificationActionsProps) => {
  const { data: doc } = useFetchDoc(docId);
  const projectId = doc?.project_id ?? "";
  const { clarify, updateDoc, toTickets } = useDocBuiltinPlays(projectId);
  const close = useCloseClarification(docId);
  const phase = clarificationPhase(clarification);
  const changed = answersChangedSinceWritten(clarification) > 0;
  const clarifyShown = phase !== "noGaps" || changed;
  const updateShown = !!updateDoc && (phase === "answered" || (phase === "noGaps" && changed));
  const clarifyPrimary = !updateShown && (phase === "none" || phase === "answered" || changed);
  const playShown = (!!clarify && clarifyShown) || updateShown || (phase === "closed" && !!toTickets);
  return (
    <span className="flex min-w-0 flex-wrap items-start gap-2">
      {updateDoc && updateShown && <PlayButton play={updateDoc} projectId={projectId} targetType="doc" targetId={docId} variant="default" />}
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
      {playShown && <HarnessReadinessNote projectId={projectId} className="basis-full font-mono" />}
    </span>
  );
};
