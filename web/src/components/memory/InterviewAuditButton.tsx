import { PlayButton } from "@/components/play/PlayButton";
import { useInterviewPlays } from "@/hooks/InterviewSourceHooks";

interface InterviewAuditButtonProps {
  projectId: string;
}

// "Audit via AI", hidden when the workspace has no audit play the caller may run on this project.
export const InterviewAuditButton = ({ projectId }: InterviewAuditButtonProps) => {
  const play = useInterviewPlays(projectId).audit;
  if (!play) return null;
  return <PlayButton play={play} projectId={projectId} targetType="interview" targetId={projectId} label="Audit via AI" className="shrink-0" />;
};
