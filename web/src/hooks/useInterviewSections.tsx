import { useMemo } from "react";

import { useFetchInterviewDrafts, useFetchInterviewSources } from "@/hooks/InterviewSourceHooks";
import { useFetchInterviewAnswers, useFetchProjectInterviewTemplate } from "@/hooks/MemoryHooks";
import { buildSections } from "@/models/InterviewAnswer";

// The interview's stored rounds: the template's questions with their answers and drafts, then each follow-up round.
export const useInterviewSections = (projectId: string) => {
  const template = useFetchProjectInterviewTemplate(projectId);
  const answers = useFetchInterviewAnswers(projectId);
  const { data: drafts } = useFetchInterviewDrafts(projectId);
  const { data: sources } = useFetchInterviewSources(projectId);
  const sections = useMemo(
    () => template.data && answers.data && buildSections(template.data.questions, answers.data, drafts, sources),
    [template.data, answers.data, drafts, sources],
  );
  return { sections, answers: answers.data, isPending: template.isPending || answers.isPending, error: template.error ?? answers.error };
};
