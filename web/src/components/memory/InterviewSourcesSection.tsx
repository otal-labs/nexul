import { useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewDraftButton } from "@/components/memory/InterviewDraftButton";
import { InterviewDraftRunLine } from "@/components/memory/InterviewDraftRunLine";
import { QuestionSection } from "@/components/questions/QuestionSection";
import { InterviewSourceList } from "@/components/memory/InterviewSourceList";
import { useFetchInterviewSources } from "@/hooks/InterviewSourceHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { sourcesMeta } from "@/models/InterviewSource";

interface InterviewSourcesSectionProps {
  projectId: string;
}

// What the interview is pointed at, foldable above the questions, with "Draft answers" and the drafting run's state.
export const InterviewSourcesSection = ({ projectId }: InterviewSourcesSectionProps) => {
  const canWrite = useHasPermission("memories:write");
  const { data: sources, error, isPending } = useFetchInterviewSources(projectId);
  const [folded, setFolded] = useState(false);
  const onlyQuestion = !!sources && sources.length > 0 && sources.every((s) => s.stance === "question");

  return (
    <QuestionSection
      label="Sources"
      meta={sources ? sourcesMeta(sources) : ""}
      folded={folded}
      onToggle={() => setFolded(!folded)}
      action={<InterviewDraftButton projectId={projectId} />}
    >
      <div className="space-y-2">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        <InterviewDraftRunLine projectId={projectId} />
        {onlyQuestion && (
          <p className="px-1.5 py-1 text-sm text-muted-foreground">Sources under question are asked about in the follow-ups, not drafted from.</p>
        )}
        {sources && <InterviewSourceList projectId={projectId} sources={sources} readOnly={!canWrite} />}
      </div>
    </QuestionSection>
  );
};
