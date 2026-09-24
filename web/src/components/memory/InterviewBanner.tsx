import { ClipboardList, X } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { useFetchMemoriesByProject } from "@/hooks/MemoryHooks";
import { useInterviewBannerStore } from "@/stores/interviewBannerStore";
import { hasInterview } from "@/models/Memory";
import { interviewPath, projectToken, type Project } from "@/models/Project";

interface InterviewBannerProps {
  project: Project;
}

// A signal, never a gate: the project works without an interview, the banner only says what agents are missing.
export const InterviewBanner = ({ project }: InterviewBannerProps) => {
  const { data: memories } = useFetchMemoriesByProject(project.id);
  const dismissed = useInterviewBannerStore((s) => s.dismissedProjectIds.includes(project.id));
  const dismiss = useInterviewBannerStore((s) => s.dismiss);

  if (!memories || dismissed || hasInterview(memories, project.id)) return null;

  return (
    <div
      role="status"
      className="animate-in fade-in-0 slide-in-from-top-1 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border border-border bg-card px-3 py-2.5 duration-200 ease-out sm:px-4"
    >
      <ClipboardList className="size-4 shrink-0 text-info" aria-hidden />
      <p className="min-w-0 flex-1 basis-48 text-sm">
        {project.name} has no interview yet, so agents here work without its rules.
      </p>
      <div className="flex items-center gap-1">
        <Button asChild size="sm" variant="outline">
          <Link to={interviewPath(projectToken(project))}>Run the interview</Link>
        </Button>
        <Button
          size="icon"
          variant="ghost"
          className="size-8"
          aria-label="Dismiss for this session"
          title="Dismiss for this session"
          onClick={() => dismiss(project.id)}
        >
          <X className="size-4" aria-hidden />
        </Button>
      </div>
    </div>
  );
};
