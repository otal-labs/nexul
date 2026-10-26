import { avatarFill } from "@nexul/client-core/avatarFill";
import { cn } from "@/lib/utils";
import { projectTile, type Project } from "@/models/Project";

interface ProjectMarkProps {
  project: Pick<Project, "prefix" | "name">;
  className?: string;
}

// A project's face: its prefix on a flat fill seeded from it, kept square by shrinking a long prefix to fit.
export const ProjectMark = ({ project, className }: ProjectMarkProps) => {
  const tile = projectTile(project);
  return (
    <span
      aria-hidden
      style={{ backgroundColor: avatarFill(project.prefix || project.name) }}
      className={cn(
        "@container flex size-11 shrink-0 items-center justify-center rounded-lg font-mono text-sm font-semibold tracking-tight text-white shadow-card select-none",
        className,
      )}
    >
      <span style={{ fontSize: `min(1em, ${Math.floor(150 / Math.max(tile.length, 1))}cqw)` }}>{tile}</span>
    </span>
  );
};
