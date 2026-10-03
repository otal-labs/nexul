import { useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { getInterviewAnswersKey } from "@/hooks/MemoryHooks";
import { useActiveTrail, useAnswerTrail } from "@/hooks/TrailHooks";
import { usePlayRunStore } from "@/stores/playRunStore";
import { liveSection, type InterviewRow, type InterviewSectionData } from "@/models/InterviewAnswer";
import type { AnswerValue, QuestionAnswers } from "@/models/Question";

// The live follow-ups of a waiting interview run: a section after the stored ones, its answers collected row by row and
// sent together on the trail once the last is given; the stored round then arrives in its place.
export const useInterviewLiveRound = (projectId: string, stored: InterviewSectionData[]) => {
  const client = useQueryClient();
  const trail = useActiveTrail("interview", projectId);
  const state = usePlayRunStore((s) => (trail ? (s.frames[trail.id]?.state ?? trail.state) : undefined));
  const question = usePlayRunStore((s) => (trail ? (s.frames[trail.id]?.question ?? trail.question ?? null) : null));
  const answerTrail = useAnswerTrail();
  const [drafts, setDrafts] = useState<{ requestId: string; answers: QuestionAnswers }>({ requestId: "", answers: {} });
  const asking = state === "waiting" && question !== null && question.answer === undefined ? question : null;
  const answers = useMemo(() => (drafts.requestId === asking?.request_id ? drafts.answers : {}), [drafts, asking?.request_id]);
  const section = useMemo(() => asking && liveSection(asking, stored, answers), [asking, stored, answers]);

  // Keeps one row's answer and returns the next row of the round to open, or sends the round when none is left.
  const answer = (row: InterviewRow, value: AnswerValue): string | null => {
    if (!trail || !asking || !section) return null;
    const next = { ...answers, [row.item.id]: value };
    setDrafts({ requestId: asking.request_id, answers: next });
    const left = section.rows.find((r) => r.item.id !== row.item.id && !(r.item.id in next));
    if (left) return left.key;
    answerTrail.mutate(
      { trailId: trail.id, answers: next },
      { onSuccess: () => void client.invalidateQueries({ queryKey: [getInterviewAnswersKey, projectId] }) },
    );
    return null;
  };

  return { section, answer, pending: answerTrail.isPending };
};
