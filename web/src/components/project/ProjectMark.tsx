import { avatarGradient } from "@/lib/avatarGradient";
import { cn } from "@/lib/utils";
import { projectTile, type Project } from "@/models/Project";

interface ProjectMarkProps {
  project: Project;
  className?: string;
}

// A project's face: its prefix on a gradient seeded from it, the same family as a person's avatar.
export const ProjectMark = ({ project, className }: ProjectMarkProps) => (
  <span
    aria-hidden
    style={{ backgroundImage: avatarGradient(project.prefix || project.name) }}
    className={cn(
      "flex size-11 shrink-0 items-center justify-center rounded-lg font-mono text-sm font-semibold tracking-tight text-white shadow-card select-none [text-shadow:0_1px_2px_oklch(0_0_0/0.35)]",
      className,
    )}
  >
    {projectTile(project)}
  </span>
);
