import { FolderGit2, X } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { useUnfinishedProject } from "@/hooks/useUnfinishedProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useUnfinishedProjectStore } from "@/stores/unfinishedProjectStore";

interface UnfinishedProjectBannerProps {
  // Shows only for this project; absent, for whichever one the wizard left unfinished.
  projectId?: string;
}

// A signal, never a gate: leaving the wizard after naming a project keeps it, and this is the way back to its repository.
export const UnfinishedProjectBanner = ({ projectId }: UnfinishedProjectBannerProps) => {
  const project = useUnfinishedProject();
  const dismiss = useUnfinishedProjectStore((s) => s.dismiss);
  const wsPath = useWorkspacePath();

  if (!project || (projectId && project.id !== projectId)) return null;

  return (
    <div
      role="status"
      className="animate-in fade-in-0 slide-in-from-top-1 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border border-border bg-card px-3 py-2.5 duration-200 ease-out sm:px-4"
    >
      <FolderGit2 className="size-4 shrink-0 text-info" aria-hidden />
      <p className="min-w-0 flex-1 basis-48 text-sm">{project.name} has no repository yet, so its setup stopped there.</p>
      <div className="flex items-center gap-1">
        <Button asChild size="sm" variant="outline">
          <Link to={wsPath(`/wizard/project/repository?project=${project.id}`)}>Continue setup</Link>
        </Button>
        <Button
          size="icon"
          variant="ghost"
          className="size-8"
          aria-label="Dismiss"
          title="Dismiss"
          onClick={dismiss}
        >
          <X className="size-4" aria-hidden />
        </Button>
      </div>
    </div>
  );
};
