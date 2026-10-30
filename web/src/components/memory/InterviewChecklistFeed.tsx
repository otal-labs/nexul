import { useState } from "react";

import { InterviewDoneRow } from "@/components/memory/InterviewDoneRow";
import { InterviewProgress } from "@/components/memory/InterviewProgress";
import { InterviewQuestionForm } from "@/components/memory/InterviewQuestionForm";
import { InterviewSuggestionCard } from "@/components/memory/InterviewSuggestionCard";
import { QuestionChecklistRow } from "@/components/questions/QuestionChecklistRow";
import { QuestionSection } from "@/components/questions/QuestionSection";
import { useInterviewLiveRound } from "@/hooks/useInterviewLiveRound";
import { useInterviewSourceFocusStore } from "@/stores/interviewSourceFocusStore";
import type { AnswerValue } from "@nexul/client-core/question";
import type { InterviewRow, InterviewSectionData } from "@/models/InterviewAnswer";
import { firstPendingKey, nextPendingKey, progressCounts, rowProgress, rowStatus } from "@nexul/client-core/questionChecklist";

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
  const railSource = useInterviewSourceFocusStore((s) => (s.focus?.from === "rail" ? s.focus.sourceId : null));
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

  const settled = (section: InterviewSectionData) => `${progressCounts(section.rows).answered}/${section.rows.length}`;

  const progressOf = (section: InterviewSectionData, i: number) =>
    section.key === "0" ? `Question ${i + 1} of ${section.rows.length}` : `Follow-up ${i + 1} of ${section.rows.length}`;

  const formProps = (section: InterviewSectionData, row: InterviewRow, i: number) => {
    const at = rows.indexOf(row);
    const progress = progressOf(section, i);
    const prevKey = rows[at - 1]?.key ?? null;
    const onAnswer = row.live && ((value: AnswerValue) => live.answer(row, value));
    return { row, projectId, progress, prevKey, nextKey: nextPendingKey(rows, row.key), onMove: move, onAnswer, pending: live.pending };
  };

  // Beside the sources the meter takes the grid's first row, so the first section and the rail start on one line.
  return (
    <div className="min-w-0 space-y-6 @min-[60rem]:row-span-2 @min-[60rem]:grid @min-[60rem]:grid-rows-subgrid @min-[60rem]:space-y-0">
      <InterviewProgress sections={sections} current={current} onJump={() => move(rows.find((r) => rowProgress(r) === "drafted" || rowProgress(r) === "waiting")?.key ?? null)} />
      <div className="min-w-0 space-y-6">
        {memoryWithoutAnswers && (
          <p className="text-sm text-muted-foreground">
            An earlier interview wrote this memory. Answer these and the agent updates it.
          </p>
        )}
        {sections.map((section) => (
          <QuestionSection
            key={section.key}
            label={section.label}
            meta={settled(section)}
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
                  lit={railSource !== null && !!row.draft?.cited.some((s) => s.id === railSource)}
                  readOnly={readOnly}
                  skippedLabel="Skipped · the agent will ask about it"
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
    </div>
  );
};
