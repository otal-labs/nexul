import { useFormContext, useWatch } from "react-hook-form";

import { ProjectAccessBlock } from "@/components/access/ProjectAccessBlock";
import { useFetchWorkspaceProjects } from "@/hooks/ProjectHooks";
import type { CreateInvitationFormData } from "@/models/Invitation";
import { accessByProject, EveryProject, withProjectGrant } from "@/models/Team";

interface InvitationProjectAccessProps {
  index: number;
}

// The Team dialog's Every project block for someone not yet here, so a client lands restricted from the first page.
export const InvitationProjectAccess = ({ index }: InvitationProjectAccessProps) => {
  const { control, setValue } = useFormContext<CreateInvitationFormData>();
  const workspaceId = useWatch({ control, name: `grants.${index}.workspace_id` });
  const everyProject = useWatch({ control, name: `grants.${index}.every_project` }) ?? EveryProject.Role;
  const grants = useWatch({ control, name: `grants.${index}.project_access` }) ?? [];
  const { data: projects } = useFetchWorkspaceProjects(workspaceId);

  return (
    <ProjectAccessBlock
      subject="the invited person"
      everyProject={everyProject}
      onEveryProject={(value) => setValue(`grants.${index}.every_project`, value)}
      projects={(projects ?? []).map((project) => ({ id: project.id, name: project.name }))}
      access={accessByProject(grants)}
      onProjectAccess={(projectId, allow) => setValue(`grants.${index}.project_access`, withProjectGrant(grants, projectId, allow))}
    />
  );
};
