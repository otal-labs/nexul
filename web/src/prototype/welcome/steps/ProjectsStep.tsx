import { FolderKanban } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { avatarGradient } from "@/lib/avatarGradient";

import { StepFooter, useStepState } from "../flow";
import { MOCK_PROJECTS, useWelcomeProto } from "../model";

const ProjectRow = ({ project }: { project: (typeof MOCK_PROJECTS)[number] }) => (
  <li className="flex items-center gap-3 py-3">
    <span
      aria-hidden
      style={{ backgroundImage: avatarGradient(project.prefix) }}
      className="grid size-8 shrink-0 place-items-center rounded-md font-mono text-[11px] font-semibold text-white shadow-card"
    >
      {project.prefix}
    </span>
    <div className="min-w-0 flex-1">
      <p className="flex items-baseline gap-2 text-sm font-medium">
        <span className="truncate">{project.name}</span>
        <span className="shrink-0 font-mono text-xs font-normal text-muted-foreground">{project.level}</span>
      </p>
      <p className="truncate text-xs text-muted-foreground">{project.can}</p>
    </div>
    <Button variant="ghost" size="sm" className="shrink-0">
      Open
    </Button>
  </li>
);

export const ProjectsStep = () => {
  const state = useStepState("projects");
  const scene = useWelcomeProto((s) => s.scene);
  const none = scene === "no-projects";
  return (
    <>
      {none && (
        <EmptyState
          size="compact"
          icon={FolderKanban}
          title="Your team hasn't added you to a project yet"
          message="Ask whoever invited you. This step ticks itself the moment they do."
        />
      )}
      {!none && (
        <ul className="divide-y divide-border border-y border-border">
          {MOCK_PROJECTS.map((p) => (
            <ProjectRow key={p.prefix} project={p} />
          ))}
        </ul>
      )}
      <StepFooter skip={none && state !== "done"} primary={none ? undefined : { label: "Continue" }} />
    </>
  );
};
