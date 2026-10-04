import { useState } from "react";

import { ClarificationRoundSection } from "@/components/doc/clarification/ClarificationRoundSection";
import { askedRounds, clarificationPhase, questionRow, type Clarification, type ClarificationRound } from "@/models/DocClarification";
import { firstPendingKey, nextPendingKey } from "@/models/QuestionChecklist";

interface ClarificationRoundsFeedProps {
  clarification: Clarification;
  readOnly: boolean;
}

// Every round that asked something, oldest first; one row is open at a time, the first unanswered on load and again
// when a new round arrives, and earlier rounds fold.
export const ClarificationRoundsFeed = ({ clarification, readOnly }: ClarificationRoundsFeedProps) => {
  const rounds = askedRounds(clarification);
  const rows = rounds.flatMap((r) => r.questions.map(questionRow));
  const newest = rounds.at(-1);
  // Chosen once, so someone else's answer arriving live never moves the row this person is in.
  const [openKey, setOpenKey] = useState<string | null>(() => firstPendingKey(rows));
  const [seenRound, setSeenRound] = useState(newest?.round);
  const [folds, setFolds] = useState<Record<number, boolean>>({});
  const closed = clarification.closed;
  const locked = readOnly || closed;
  const arrived = newest !== undefined && newest.round !== seenRound;
  const opened = arrived ? firstPendingKey(newest.questions.map(questionRow)) : openKey;
  const current = locked ? null : opened;
  // The newest round takes its "Anything else?" until the next round runs or one finds no gaps.
  const phase = clarificationPhase(clarification);
  const answering = phase === "waiting" || phase === "answered";

  const folded = (r: ClarificationRound) =>
    folds[r.round] ?? (closed || (r !== newest && !r.questions.some((q) => q.id === current)));

  const move = (key: string | null) => {
    setOpenKey(key);
    setSeenRound(newest?.round);
    const round = rounds.find((r) => r.questions.some((q) => q.id === key));
    if (round) setFolds((f) => ({ ...f, [round.round]: false }));
  };

  const around = (key: string) => {
    const at = rows.findIndex((r) => r.key === key);
    return { prevKey: rows[at - 1]?.key ?? null, nextKey: nextPendingKey(rows, key) };
  };

  return (
    <div className="mt-5 space-y-6">
      {rounds.map((round) => (
        <ClarificationRoundSection
          key={round.round}
          round={round}
          current={current}
          readOnly={locked}
          folded={folded(round)}
          open={round === newest && !locked && answering}
          onToggleFold={() => setFolds((f) => ({ ...f, [round.round]: !folded(round) }))}
          onMove={move}
          around={around}
        />
      ))}
    </div>
  );
};
