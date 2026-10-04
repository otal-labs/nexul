import { useState } from "react";

import { InterviewDoneRow } from "@/components/memory/InterviewDoneRow";
import { InterviewQuestionForm } from "@/components/memory/InterviewQuestionForm";
import { InterviewSuggestionCard } from "@/components/memory/InterviewSuggestionCard";
import { QuestionChecklistRow } from "@/components/questions/QuestionChecklistRow";
import { QuestionSection } from "@/components/questions/QuestionSection";
import { useInterviewLiveRound } from "@/hooks/useInterviewLiveRound";
import type { AnswerValue } from "@/models/Question";
import type { InterviewRow, InterviewSectionData } from "@/models/InterviewAnswer";
import { countLine, firstPendingKey, nextPendingKey, rowStatus } from "@/models/QuestionChecklist";

interface InterviewChecklistFeedProps {
  projectId: string;
  sections: InterviewSectionData[];
  readOnly: boolean;
  // With a memory, the memory column's Regenerate starts the run; without one, Done does, also after a run that failed.
  hasMemory: boolean;
  // The project has a memory from before its questions were answered.
  memoryWithoutAnswers: boolean;
}

// The questions as numbered rows in foldable sections; one row is open at a time, the first unanswered on load.
export const InterviewChecklistFeed = ({ projectId, sections: stored, readOnly, hasMemory, memoryWithoutAnswers }: InterviewChecklistFeedProps) => {
  const live = useInterviewLiveRound(projectId, stored);
  const sections = live.section ? [...stored, live.section] : stored;
  const rows = sections.flatMap((s) => s.rows);
  // Chosen once, so someone else's answer arriving live never moves the row this person is typing in.
  const [openKey, setOpenKey] = useState<string | null>(() => firstPendingKey(rows));
  // A new live round opens its first question once, until this person moves.
  const [openFor, setOpenFor] = useState<string | undefined>();
  const [folds, setFolds] = useState<Record<string, boolean>>({});
  const roundArrived = live.section !== null && openFor !== live.section.key;
  const opened = roundArrived ? firstPendingKey(live.section?.rows ?? []) : openKey;
  const current = readOnly ? null : opened;
  const newest = sections.at(-1)?.key;
  const initial = sections[0];
  const templateDone = initial !== undefined && initial.rows.every((r) => rowStatus(r) !== "pending");

  const folded = (section: InterviewSectionData) =>
    folds[section.key] ?? (section.key !== newest && !section.rows.some((r) => r.key === current));

  const move = (key: string | null) => {
    setOpenKey(key);
    setOpenFor(live.section?.key);
    const section = sections.find((s) => s.rows.some((r) => r.key === key));
    if (section) setFolds((f) => ({ ...f, [section.key]: false }));
  };

  const progressOf = (section: InterviewSectionData, i: number) =>
    section.key === "0" ? `Question ${i + 1} of ${section.rows.length}` : `Follow-up ${i + 1} of ${section.rows.length}`;

  const formProps = (section: InterviewSectionData, row: InterviewRow, i: number) => {
    const at = rows.indexOf(row);
    const progress = progressOf(section, i);
    const prevKey = rows[at - 1]?.key ?? null;
    const onAnswer = row.live && ((value: AnswerValue) => live.answer(row, value));
    return { row, projectId, progress, prevKey, nextKey: nextPendingKey(rows, row.key), onMove: move, onAnswer, pending: live.pending };
  };

  return (
    <div className="min-w-0 space-y-6">
      {memoryWithoutAnswers && (
        <p className="text-sm text-muted-foreground">
          This memory came from an earlier interview. Answering these questions lets the agent update it.
        </p>
      )}
      {sections.map((section) => (
        <QuestionSection
          key={section.key}
          label={section.label}
          meta={countLine(section.rows)}
          folded={folded(section)}
          onToggle={() => setFolds((f) => ({ ...f, [section.key]: !folded(section) }))}
        >
          <ol>
            {section.rows.map((row, i) => (
              <QuestionChecklistRow
                key={row.key}
                row={row}
                number={rows.indexOf(row) + 1}
                open={row.key === current}
                readOnly={readOnly}
                onToggle={() => move(row.key === current ? null : row.key)}
              >
                {row.draft?.state === "suggested" && (
                  <InterviewSuggestionCard row={row} draft={row.draft} projectId={projectId} progress={progressOf(section, i)} onMove={move} />
                )}
                {/* Keyed on the draft so one that lands while its card is open seeds the form; confirming keeps the key. */}
                {row.draft?.state !== "suggested" && <InterviewQuestionForm key={row.draft?.id ?? "none"} {...formProps(section, row, i)} />}
              </QuestionChecklistRow>
            ))}
          </ol>
        </QuestionSection>
      ))}
      {templateDone && !hasMemory && !live.section && <InterviewDoneRow projectId={projectId} />}
    </div>
  );
};
