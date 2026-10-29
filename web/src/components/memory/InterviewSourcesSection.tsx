import { useState } from "react";
import { Plus } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewDraftButton } from "@/components/memory/InterviewDraftButton";
import { InterviewDraftRunLine } from "@/components/memory/InterviewDraftRunLine";
import { InterviewSourceAddForm } from "@/components/memory/InterviewSourceAddForm";
import { InterviewSourceList } from "@/components/memory/InterviewSourceList";
import { useFetchInterviewSources } from "@/hooks/InterviewSourceHooks";
import { useActionState } from "@/hooks/useActionState";

interface InterviewSourcesSectionProps {
  projectId: string;
}

// What the interview drafts from and asks about, beside the questions from 60rem (its own scroll, pinned to the panel's top)
// and under them below that.
export const InterviewSourcesSection = ({ projectId }: InterviewSourcesSectionProps) => {
  const canWrite = useActionState({ permission: "memories:write", projectId }).state === "live";
  const { data: sources, error, isPending } = useFetchInterviewSources(projectId);
  const [adding, setAdding] = useState(false);
  const onlyQuestion = !!sources && sources.length > 0 && sources.every((s) => s.stance === "question");

  return (
    <section
      aria-label="Sources"
      data-source-rail
      className="min-w-0 space-y-4 @min-[60rem]:sticky @min-[60rem]:top-6 @min-[60rem]:-mx-2 @min-[60rem]:max-h-[calc(100vh-3rem)] @min-[60rem]:overflow-y-auto @min-[60rem]:px-2 @min-[60rem]:pb-6 @min-[60rem]:[mask-image:linear-gradient(to_bottom,#000_calc(100%-24px),transparent)]"
    >
      <header className="space-y-1.5 border-b border-border pb-2">
        <div className="flex min-h-8 items-center gap-2">
          <h2 className="flex flex-1 items-baseline gap-2 px-1.5 text-sm font-semibold">
            Sources
            {sources && sources.length > 0 && <span className="font-mono text-xs font-normal text-muted-foreground tabular-nums">{sources.length}</span>}
          </h2>
          <InterviewDraftButton projectId={projectId} />
          <ActionButton
            permission="memories:write"
            projectId={projectId}
            verb="add sources"
            icon={Plus}
            variant="ghost"
            size="icon"
            className="size-8 shrink-0 text-muted-foreground"
            aria-label="Add source"
            title="Add source"
            onClick={() => setAdding(true)}
          />
        </div>
        <InterviewDraftRunLine projectId={projectId} />
      </header>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {adding && <InterviewSourceAddForm projectId={projectId} onClose={() => setAdding(false)} />}
      {onlyQuestion && <p className="px-1.5 text-xs text-muted-foreground">Sources under question are asked about in the follow-ups, not drafted from.</p>}
      {sources && <InterviewSourceList projectId={projectId} sources={sources} readOnly={!canWrite} />}
    </section>
  );
};
