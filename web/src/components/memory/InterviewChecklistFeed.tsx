import { useState } from "react";

import { InterviewDoneRow } from "@/components/memory/InterviewDoneRow";
import { InterviewQuestionForm } from "@/components/memory/InterviewQuestionForm";
import { InterviewQuestionRow } from "@/components/memory/InterviewQuestionRow";
import { InterviewSection } from "@/components/memory/InterviewSection";
import { countLine, firstPendingKey, nextPendingKey, rowStatus, type InterviewRow, type InterviewSectionData } from "@/models/InterviewAnswer";

interface InterviewChecklistFeedProps {
  projectId: string;
  sections: InterviewSectionData[];
  readOnly: boolean;
  // The project has a memory from before its questions were answered.
  memoryWithoutAnswers: boolean;
}

// The questions as numbered rows in foldable sections; one row is open at a time, the first unanswered on load.
export const InterviewChecklistFeed = ({ projectId, sections, readOnly, memoryWithoutAnswers }: InterviewChecklistFeedProps) => {
  const rows = sections.flatMap((s) => s.rows);
  // Chosen once, so someone else's answer arriving live never moves the row this person is typing in.
  const [openKey, setOpenKey] = useState<string | null>(() => firstPendingKey(rows));
  const [folds, setFolds] = useState<Record<string, boolean>>({});
  const current = readOnly ? null : openKey;
  const newest = sections.at(-1)?.key;
  const initial = sections[0];
  const templateDone = initial !== undefined && initial.rows.every((r) => rowStatus(r) !== "pending");

  const folded = (section: InterviewSectionData) =>
    folds[section.key] ?? (section.key !== newest && !section.rows.some((r) => r.key === current));

  const move = (key: string | null) => {
    setOpenKey(key);
    const section = sections.find((s) => s.rows.some((r) => r.key === key));
    if (section) setFolds((f) => ({ ...f, [section.key]: false }));
  };

  const formProps = (section: InterviewSectionData, row: InterviewRow, i: number) => {
    const at = rows.indexOf(row);
    const progress = section.key === "0" ? `Question ${i + 1} of ${section.rows.length}` : `Follow-up ${i + 1} of ${section.rows.length}`;
    return { row, projectId, progress, prevKey: rows[at - 1]?.key ?? null, nextKey: nextPendingKey(rows, row.key), onMove: move };
  };

  return (
    <div className="min-w-0 space-y-6">
      {memoryWithoutAnswers && (
        <p className="text-sm text-muted-foreground">
          This memory came from an earlier interview. Answering these questions lets the agent update it.
        </p>
      )}
      {sections.map((section) => (
        <InterviewSection
          key={section.key}
          label={section.label}
          meta={countLine(section.rows)}
          folded={folded(section)}
          onToggle={() => setFolds((f) => ({ ...f, [section.key]: !folded(section) }))}
        >
          <ol>
            {section.rows.map((row, i) => (
              <InterviewQuestionRow
                key={row.key}
                row={row}
                number={rows.indexOf(row) + 1}
                open={row.key === current}
                readOnly={readOnly}
                onToggle={() => move(row.key === current ? null : row.key)}
              >
                <InterviewQuestionForm {...formProps(section, row, i)} />
              </InterviewQuestionRow>
            ))}
          </ol>
        </InterviewSection>
      ))}
      {templateDone && sections.length === 1 && <InterviewDoneRow projectId={projectId} />}
    </div>
  );
};
