import { PlayButton } from "@/components/play/PlayButton";
import { useFetchApplicablePlays } from "@/hooks/PlayHooks";
import { DRAFT_INTERVIEW_KEY } from "@/models/Play";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface InterviewRunButtonProps {
  projectId: string;
  label: string;
  variant?: "outline" | "default";
}

// Hidden when the workspace has no enabled interview play, besides the drafting one, the caller may run on this project.
export const InterviewRunButton = ({ projectId, label, variant = "outline" }: InterviewRunButtonProps) => {
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: plays } = useFetchApplicablePlays(workspaceId, projectId, "interview", undefined);
  const play = plays?.find((p) => p.builtin_key !== DRAFT_INTERVIEW_KEY);

  if (!play) return null;

  return (
    <PlayButton play={play} projectId={projectId} targetType="interview" targetId={projectId} variant={variant} label={label} />
  );
};
