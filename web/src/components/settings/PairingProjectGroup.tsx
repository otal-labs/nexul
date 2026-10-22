import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PairingProjectRow } from "@/components/settings/PairingProjectRow";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { cn } from "@/lib/utils";
import type { Workspace } from "@/models/Workspace";

interface PairingProjectGroupProps {
  workspace: Workspace;
  openId: string | undefined;
  onToggle: (projectId: string) => void;
}

// One workspace's projects, set into the shared list with no heading of their own; a workspace with none adds nothing.
export const PairingProjectGroup = ({ workspace, openId, onToggle }: PairingProjectGroupProps) => {
  const { data: projects, isPending, error } = useFetchWorkspaceProjects(workspace.id);

  return (
    <li className={cn(projects?.length === 0 && "hidden")}>
      {isPending && <LoadingDisplay className="justify-start px-3 py-2.5" />}
      {error && <ErrorDisplay error={error} />}
      {projects && projects.length > 0 && (
        <ul aria-label={`${workspace.name} projects`} className="divide-y divide-border">
          {projects.map((project) => (
            <PairingProjectRow
              key={project.id}
              project={project}
              open={openId === project.id}
              onToggle={() => onToggle(project.id)}
            />
          ))}
        </ul>
      )}
    </li>
  );
};
