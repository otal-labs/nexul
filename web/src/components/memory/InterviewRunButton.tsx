import { PlayButton } from "@/components/play/PlayButton";
import { useInterviewPlays } from "@/hooks/InterviewSourceHooks";

interface InterviewRunButtonProps {
  projectId: string;
  label: string;
  variant?: "outline" | "default";
}

// Hidden when the workspace has no enabled follow-up interview play the caller may run on this project.
export const InterviewRunButton = ({ projectId, label, variant = "outline" }: InterviewRunButtonProps) => {
  const play = useInterviewPlays(projectId).followUp;

  if (!play) return null;

  return (
    <PlayButton play={play} projectId={projectId} targetType="interview" targetId={projectId} variant={variant} label={label} />
  );
};
