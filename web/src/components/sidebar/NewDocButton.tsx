import { PlusIcon } from "lucide-react";

import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import type { Project } from "@/models/Project";

interface NewDocButtonProps {
  project: Project;
}

// Revealed by hovering the project switcher row (group/project).
export const NewDocButton = ({ project }: NewDocButtonProps) => {
  const createDoc = useCreateDocDialog(project.id);

  return (
    <button
      type="button"
      onClick={createDoc}
      aria-label={`New doc in ${project.name}`}
      className="shrink-0 rounded p-1 text-muted-foreground opacity-0 transition-opacity duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:opacity-100 group-hover/project:opacity-100"
    >
      <PlusIcon className="size-3.5" aria-hidden />
    </button>
  );
};
