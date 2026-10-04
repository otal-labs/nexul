import { useShallow } from "zustand/react/shallow";

import { InterviewQuestionRow } from "@/components/memory/InterviewQuestionRow";
import { InterviewSection } from "@/components/memory/InterviewSection";
import type { ProtoQuestion } from "@/components/doc/prototype/ClarifyPrototypeData";
import { ClarifyPrototypeNoteBox, ClarifyPrototypeNoteReply } from "@/components/doc/prototype/ClarifyPrototypeNote";
import { ClarifyPrototypeQuestionForm } from "@/components/doc/prototype/ClarifyPrototypeQuestionForm";
import { roundQuestions, useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import { countLine, type InterviewRow } from "@/models/InterviewAnswer";
import type { AnswerValue } from "@/models/Question";

interface ClarifyPrototypeRoundProps {
  n: number;
}

const toRow = (item: ProtoQuestion, round: number, value: AnswerValue | undefined, skipped: boolean): InterviewRow => ({
  key: item.id,
  round,
  item,
  why: item.why,
  answer:
    value || skipped
      ? {
          id: item.id, workspace_id: "", project_id: "", round, question: item.text,
          selected: value?.selected ?? [], text: value?.text ?? "", skipped, answered_by: "", answered_at: "",
        }
      : undefined,
});

// One round as the interview's foldable section of numbered rows; the newest open round ends with "Anything else?".
export const ClarifyPrototypeRound = ({ n }: ClarifyPrototypeRoundProps) => {
  const s = useClarifyPrototypeStore(
    useShallow((st) => ({
      answers: st.answers, skipped: st.skipped, openId: st.openId, folds: st.folds,
      rounds: st.rounds, phase: st.phase, open: st.open, toggleFold: st.toggleFold,
    })),
  );
  const rows = roundQuestions(n).map((q) => toRow(q, n, s.answers[q.id], s.skipped.includes(q.id)));
  const newest = n === s.rounds;
  const live = newest && s.phase === "open";
  const closed = s.phase === "closed";
  const folded = s.folds[n] ?? (closed || (!newest && !rows.some((r) => r.key === s.openId)));

  return (
    <div>
      <InterviewSection label={`Round ${n}`} meta={countLine(rows)} folded={folded} onToggle={() => s.toggleFold(n, folded)}>
        <ol>
          {rows.map((row, i) => (
            <InterviewQuestionRow
              key={row.key}
              row={row}
              number={i + 1}
              open={row.key === s.openId}
              readOnly={closed}
              onToggle={() => s.open(row.key === s.openId ? null : row.key)}
            >
              <ClarifyPrototypeQuestionForm item={row.item} why={row.why} progress={`Question ${i + 1} of ${rows.length}`} first={n === 1 && i === 0} />
            </InterviewQuestionRow>
          ))}
        </ol>
        {live && <ClarifyPrototypeNoteBox round={n} />}
      </InterviewSection>
      {!live && <ClarifyPrototypeNoteReply round={n} />}
    </div>
  );
};
