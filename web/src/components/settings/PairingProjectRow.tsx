import { ChevronDownIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ProjectLinkForm } from "@/components/settings/ProjectLinkForm";
import { useListComputers } from "@/hooks/PairingHooks";
import { useFetchProjectLinks, useProjectLinkSummary } from "@/hooks/PairingProjectHooks";
import type { Project } from "@/models/Project";
import { cn } from "@/lib/utils";

interface PairingProjectRowProps {
  project: Project;
  open: boolean;
  onToggle: () => void;
}

// A permission-style row: the project, what the caller's turns there run on, and a chevron opening their link inline.
export const PairingProjectRow = ({ project, open, onToggle }: PairingProjectRowProps) => {
  const { data: computers } = useListComputers();
  const { data: links } = useFetchProjectLinks();
  const link = links?.find((l) => l.project_id === project.id);
  const summary = useProjectLinkSummary(link);

  return (
    <li>
      <div className="flex min-h-11 items-center gap-3 py-1.5">
        <p className="min-w-0 flex-1 truncate text-sm">{project.name}</p>
        <p className="min-w-0 max-w-[60%] truncate font-mono text-xs text-muted-foreground">{summary}</p>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0 text-muted-foreground"
          aria-expanded={open}
          aria-label={`${open ? "Close" : "Open"} ${project.name}'s link`}
          onClick={onToggle}
        >
          <ChevronDownIcon className={cn("size-4 transition-transform duration-150 ease-standard", open && "rotate-180")} aria-hidden />
        </Button>
      </div>
      {open && computers && (
        <div className="animate-in fade-in-0 slide-in-from-top-1 border-t border-border pt-4 pb-5 pl-5 duration-150 ease-out">
          <ProjectLinkForm projectId={project.id} projectName={project.name} link={link ?? {}} computers={computers} />
        </div>
      )}
    </li>
  );
};
