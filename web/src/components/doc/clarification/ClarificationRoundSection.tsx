import { AnythingElseBox } from "@/components/doc/clarification/AnythingElseBox";
import { AnythingElseReply } from "@/components/doc/clarification/AnythingElseReply";
import { ClarificationQuestionForm } from "@/components/doc/clarification/ClarificationQuestionForm";
import { QuestionChecklistRow } from "@/components/questions/QuestionChecklistRow";
import { QuestionSection } from "@/components/questions/QuestionSection";
import { questionRow, type ClarificationRound } from "@/models/Clarification";
import { countLine } from "@/models/QuestionChecklist";

interface ClarificationRoundSectionProps {
  round: ClarificationRound;
  current: string | null;
  readOnly: boolean;
  folded: boolean;
  // The round still taking its "Anything else?"; an earlier one shows what was written and the reply instead.
  open: boolean;
  onToggleFold: () => void;
  onMove: (key: string | null) => void;
  around: (key: string) => { prevKey: string | null; nextKey: string | null };
}

// One round as a foldable section of numbered rows, ending with its "Anything else?".
export const ClarificationRoundSection = ({ round, current, readOnly, folded, open, onToggleFold, onMove, around }: ClarificationRoundSectionProps) => {
  return (
    <div>
      <QuestionSection label={`Round ${round.round}`} meta={countLine(round.questions.map(questionRow))} folded={folded} onToggle={onToggleFold}>
        <ol>
          {round.questions.map((question, i) => (
            <QuestionChecklistRow
              key={question.id}
              row={questionRow(question)}
              number={i + 1}
              open={question.id === current}
              readOnly={readOnly}
              onToggle={() => onMove(question.id === current ? null : question.id)}
            >
              <ClarificationQuestionForm
                question={question}
                progress={`Question ${i + 1} of ${round.questions.length}`}
                onMove={onMove}
                {...around(question.id)}
              />
            </QuestionChecklistRow>
          ))}
        </ol>
        {open && <AnythingElseBox round={round} />}
      </QuestionSection>
      {!open && <AnythingElseReply round={round} />}
    </div>
  );
};
