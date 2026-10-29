import { NotebookPen } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";

import { EmptyState } from "@/components/EmptyState";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InterviewMemoryView } from "@/components/memory/InterviewMemoryView";
import { TrailSection } from "@/components/play/TrailSection";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useInterviewLiveRound } from "@/hooks/useInterviewLiveRound";
import { useInterviewSections } from "@/hooks/useInterviewSections";
import { useTabPath } from "@/hooks/useTabPath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { isWrittenInterview } from "@/models/Memory";
import { projectToken, type Project } from "@/models/Project";
import { progressCounts } from "@nexul/client-core/questionChecklist";

interface InterviewMemorySectionProps {
  project: Project;
}

export const InterviewMemorySection = ({ project }: InterviewMemorySectionProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: memories, error, isPending } = useFetchMemoriesByProject(project.id);
  const memory = memories?.find((m) => isWrittenInterview(m, project.id));
  const { tabPath } = useTabPath();
  const stored = useInterviewSections(project.id).sections;
  const live = useInterviewLiveRound(project.id, stored ?? []).section;
  const needsYou = progressCounts([...(stored ?? []), ...(live ? [live] : [])].flatMap((s) => s.rows)).needsYou;

  return (
    <div className="grid items-start gap-x-12 gap-y-10 @min-[60rem]:grid-cols-[minmax(0,1fr)_20rem]">
      <section aria-label="Interview memory" className="min-w-0 space-y-4">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {memories && !memory && (
          <EmptyState
            size="compact"
            icon={NotebookPen}
            title="No memory yet"
            message={
              needsYou > 0
                ? `${needsYou} ${needsYou === 1 ? "question needs" : "questions need"} you first; then the agent writes the memory here.`
                : "The agent writes the memory here once its follow-ups are answered."
            }
            action={
              <Button asChild variant="outline" size="sm">
                <Link to={tabPath()}>Answer next</Link>
              </Button>
            }
          />
        )}
        {memory && <InterviewMemoryView memory={memory} projectToken={projectToken(project)} />}
      </section>
      <TrailSection
        workspaceId={workspaceId}
        targetType="interview"
        targetId={project.id}
        rowLayout="stacked"
        className="min-w-0 @min-[60rem]:border-t-0 @min-[60rem]:pt-1"
      />
    </div>
  );
};
