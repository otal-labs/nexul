import { ConversationThread } from "@/components/chat/ConversationThread";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchOrCreateInterviewThread } from "@/hooks/ChatHooks";
import { useFetchTrails } from "@/hooks/TrailHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface InterviewThreadSectionProps {
  projectId: string;
}

// The thread the Interview play asks its questions in; it exists once a run has created it, so none shows before.
export const InterviewThreadSection = ({ projectId }: InterviewThreadSectionProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: trails } = useFetchTrails("interview", projectId);
  const hasRun = !!trails && trails.length > 0;
  const { data: conversation, error, isPending } = useFetchOrCreateInterviewThread(workspaceId, projectId, hasRun);

  if (!hasRun) return null;

  return (
    <section className="space-y-3">
      <h2 className="text-sm font-semibold tracking-tight">Conversation</h2>
      {isPending && <LoadingDisplay label="Loading the conversation…" />}
      {error && <ErrorDisplay error={error} title="Failed to load the conversation." />}
      {conversation && (
        <div className="h-96 overflow-hidden rounded-lg border border-border">
          <ConversationThread workspaceId={workspaceId} conversation={conversation} showHeader={false} />
        </div>
      )}
    </section>
  );
};
