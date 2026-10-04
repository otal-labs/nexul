import { CircleAlert } from "lucide-react";

import { useFetchDocClarification } from "@/hooks/DocHooks";
import { unansweredInNewestRound } from "@/models/Clarification";

interface UnansweredQuestionsSignalProps {
  docId: string;
}

// Says the doc's newest round still waits on answers before another round starts; a signal, never a block.
export const UnansweredQuestionsSignal = ({ docId }: UnansweredQuestionsSignalProps) => {
  const { data } = useFetchDocClarification(docId);
  const unanswered = data && unansweredInNewestRound(data);
  if (!unanswered || unanswered.count === 0) return null;
  const noun = unanswered.count === 1 ? "question" : "questions";
  const verb = unanswered.count === 1 ? "is" : "are";
  return (
    <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
      <CircleAlert className="size-4 shrink-0 text-warning" aria-hidden />
      {unanswered.count} {noun} in Round {unanswered.round} {verb} still unanswered
    </p>
  );
};
