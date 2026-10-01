import type { ReactNode } from "react";

import { ProjectRevokedState } from "@/components/project/ProjectRevokedState";
import { useRevokedProject } from "@/hooks/ProjectHooks";

interface ProjectRevokedGateProps {
  projectId: string | undefined;
  children: ReactNode;
}

export const ProjectRevokedGate = ({ projectId, children }: ProjectRevokedGateProps) => {
  const revoked = useRevokedProject(projectId);
  return (
    <>
      {revoked && <ProjectRevokedState />}
      {!revoked && children}
    </>
  );
};
