import { ProjectAccessBlock } from "@/components/access/ProjectAccessBlock";
import { useProjectAreas } from "@/hooks/PermissionHooks";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import { useSetTeamEveryProject, useSetTeamProjectAccess } from "@/hooks/TeamHooks";
import { changeSummary } from "@/models/PermissionLevel";
import { EveryProject, type TeamMembership, type TeamPerson, type TeamWorkspace } from "@/models/Team";
import { personName } from "@/utils/TeamUtility";

interface TeamProjectAccessProps {
  person: TeamPerson;
  workspace: TeamWorkspace;
  membership: TeamMembership;
}

// A manager picks from every project they may open; a read-only viewer sees only the projects already given.
export const TeamProjectAccess = ({ person, workspace, membership }: TeamProjectAccessProps) => {
  const editable = workspace.can_manage_members;
  const { data: workspaceProjects } = useFetchWorkspaceProjects(workspace.id, editable);
  const areas = useProjectAreas();
  const setEveryProject = useSetTeamEveryProject();
  const setProjectAccess = useSetTeamProjectAccess();
  const name = personName(person);
  const target = { workspaceId: workspace.id, userId: person.id };
  const given = membership.projects.map((grant) => ({ id: grant.project_id, name: grant.project_name ?? grant.project_id }));
  const projects = (editable && workspaceProjects?.map((project) => ({ id: project.id, name: project.name }))) || given;
  const access = Object.fromEntries(membership.projects.map((grant) => [grant.project_id, grant.allow]));

  const onEveryProject = (everyProject: EveryProject) =>
    setEveryProject.mutate({
      ...target,
      everyProject,
      message:
        everyProject === EveryProject.None
          ? `${name} now sees only the ${workspace.name} projects you choose`
          : `${name} sees every ${workspace.name} project through their role`,
    });

  const onProjectAccess = (projectId: string, allow: string[]) => {
    const project = projects.find((candidate) => candidate.id === projectId)?.name ?? projectId;
    const message = allow.length === 0 ? `${name} no longer sees ${project}` : `${name} · ${project}: ${changeSummary(areas, access[projectId] ?? [], allow)}`;
    setProjectAccess.mutate({ ...target, projectId, allow, message });
  };

  return (
    <ProjectAccessBlock
      subject={name}
      everyProject={membership.every_project}
      onEveryProject={onEveryProject}
      projects={projects}
      access={access}
      onProjectAccess={onProjectAccess}
      disabled={!editable}
    />
  );
};
