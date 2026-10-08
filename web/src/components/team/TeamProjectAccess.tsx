import { Microheader } from "@/components/Microheader";
import { ProjectAccessBlock } from "@/components/access/ProjectAccessBlock";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { useMemberDraft } from "@/hooks/useMemberDraft";
import { accessByProject, type TeamMembership, type TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamProjectAccessProps {
  workspace: TeamWorkspace;
  membership: TeamMembership;
}

// A manager picks from every project they may open; a read-only viewer sees only the projects already given.
export const TeamProjectAccess = ({ workspace, membership }: TeamProjectAccessProps) => {
  const { person, dispatch } = useMemberDraft();
  const editable = workspace.can_manage_members;
  const { data: workspaceProjects } = useFetchWorkspaceProjects(workspace.id, editable);
  const given = membership.projects.map((grant) => ({ id: grant.project_id, name: grant.project_name ?? grant.project_id }));
  const projects = (editable && workspaceProjects?.map((project) => ({ id: project.id, name: project.name }))) || given;

  return (
    <section className="space-y-2">
      <Microheader>Projects</Microheader>
      <ProjectAccessBlock
        subject={personName(person)}
        everyProject={membership.every_project}
        onEveryProject={(everyProject) => dispatch({ type: "change", workspaceId: workspace.id, change: { everyProject } })}
        projects={projects}
        access={accessByProject(membership.projects)}
        onProjectAccess={(projectId, allow) => dispatch({ type: "project", workspaceId: workspace.id, projectId, allow })}
        disabled={!editable}
      />
    </section>
  );
};
