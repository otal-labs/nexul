import { InterviewRunButton } from "@/components/memory/InterviewRunButton";
import { useActiveTrail } from "@/hooks/TrailHooks";

interface InterviewDoneRowProps {
  projectId: string;
}

// Shown once every initial question is answered or skipped; gone while a run is under way, whose header shows it.
export const InterviewDoneRow = ({ projectId }: InterviewDoneRowProps) => {
  const active = useActiveTrail("interview", projectId);
  if (active) return null;
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="min-w-0 flex-1 text-sm text-muted-foreground">
        That's every question. The agent reads your answers and the code, asks about any gaps, then writes the memory.
      </p>
      <InterviewRunButton projectId={projectId} label="Done" variant="default" />
    </div>
  );
};
