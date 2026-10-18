import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewDraftButton } from "@/components/memory/InterviewDraftButton";
import { InterviewDraftRunLine } from "@/components/memory/InterviewDraftRunLine";
import { InterviewSourceList } from "@/components/memory/InterviewSourceList";
import { useFetchInterviewSources } from "@/hooks/InterviewSourceHooks";
import { useActionState } from "@/hooks/useActionState";
import { sourcesMeta } from "@/models/InterviewSource";

interface InterviewSourcesSectionProps {
  projectId: string;
}

// What the interview is pointed at, with "Draft answers" and the drafting run's state.
export const InterviewSourcesSection = ({ projectId }: InterviewSourcesSectionProps) => {
  const canWrite = useActionState({ permission: "memories:write", projectId }).state === "live";
  const { data: sources, error, isPending } = useFetchInterviewSources(projectId);
  const onlyQuestion = !!sources && sources.length > 0 && sources.every((s) => s.stance === "question");

  return (
    <section aria-label="Sources" className="space-y-2">
      <header className="flex min-h-9 flex-wrap items-center justify-between gap-2 border-b border-border pb-1">
        <p className="px-1.5 font-mono text-xs text-muted-foreground tabular-nums">{sources ? sourcesMeta(sources) : ""}</p>
        <InterviewDraftButton projectId={projectId} />
      </header>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      <InterviewDraftRunLine projectId={projectId} />
      {onlyQuestion && (
        <p className="px-1.5 py-1 text-sm text-muted-foreground">Sources under question are asked about in the follow-ups, not drafted from.</p>
      )}
      {sources && <InterviewSourceList projectId={projectId} sources={sources} readOnly={!canWrite} />}
    </section>
  );
};
