import { useState } from "react";

import { ClarificationRoundSection } from "@/components/doc/clarification/ClarificationRoundSection";
import { askedRounds, questionRow, startsFolded, takesAnythingElse, type Clarification, type ClarificationRound } from "@nexul/client-core/clarification";
import { firstPendingKey, keysAround } from "@nexul/client-core/questionChecklist";

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

  const folded = (r: ClarificationRound) => folds[r.round] ?? startsFolded(clarification, r, current);

  const move = (key: string | null) => {
    setOpenKey(key);
    setSeenRound(newest?.round);
    const round = rounds.find((r) => r.questions.some((q) => q.id === key));
    if (round) setFolds((f) => ({ ...f, [round.round]: false }));
  };

  return (
    <div className="mt-5 space-y-6">
      {rounds.map((round, i) => (
        <ClarificationRoundSection
          key={round.round}
          round={round}
          number={i + 1}
          current={current}
          readOnly={locked}
          folded={folded(round)}
          open={!locked && takesAnythingElse(clarification, round)}
          onToggleFold={() => setFolds((f) => ({ ...f, [round.round]: !folded(round) }))}
          onMove={move}
          around={(key) => keysAround(rows, key)}
        />
      ))}
    </div>
  );
};
