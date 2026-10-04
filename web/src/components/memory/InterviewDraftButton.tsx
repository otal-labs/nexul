import { PlayButton } from "@/components/play/PlayButton";
import { useFetchInterviewSources, useInterviewPlays, useInterviewTrails } from "@/hooks/InterviewSourceHooks";
import { sourcesChangedSince } from "@/models/InterviewSource";

interface InterviewDraftButtonProps {
  projectId: string;
}

// "Draft answers" once a follow source exists, with the warning dot when a source is new or changed since the last drafting.
export const InterviewDraftButton = ({ projectId }: InterviewDraftButtonProps) => {
  const play = useInterviewPlays(projectId).draft;
  const { data: sources } = useFetchInterviewSources(projectId);
  const drafting = useInterviewTrails(projectId).drafting;
  if (!play || !sources?.some((s) => s.stance === "follow")) return null;
  return (
    <PlayButton
      play={play}
      projectId={projectId}
      targetType="interview"
      targetId={projectId}
      label="Draft answers"
      outOfDate={sourcesChangedSince(sources, drafting ?? [])}
      className="shrink-0"
    />
  );
};
