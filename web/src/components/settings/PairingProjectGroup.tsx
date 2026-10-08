import { Microheader } from "@/components/access/Microheader";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { PairingProjectRow } from "@/components/settings/PairingProjectRow";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import type { Workspace } from "@/models/Workspace";

interface PairingProjectGroupProps {
  workspace: Workspace;
  // Named only when the person is in more than one workspace.
  labelled: boolean;
  openId: string | undefined;
  onToggle: (projectId: string) => void;
}

export const PairingProjectGroup = ({ workspace, labelled, openId, onToggle }: PairingProjectGroupProps) => {
  const { data: projects, isPending, error } = useFetchWorkspaceProjects(workspace.id);

  return (
    <section aria-label={`${workspace.name} projects`}>
      {labelled && <Microheader className="mb-1.5">{workspace.name}</Microheader>}
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {projects && projects.length === 0 && (
        <EmptyRow className="rounded-none border-x-0 border-b-0 px-0 py-3 text-left">No projects you can open here.</EmptyRow>
      )}
      {projects && projects.length > 0 && (
        <ul className="divide-y divide-border border-t border-border">
          {projects.map((project) => (
            <PairingProjectRow key={project.id} project={project} open={openId === project.id} onToggle={() => onToggle(project.id)} />
          ))}
        </ul>
      )}
    </section>
  );
};
