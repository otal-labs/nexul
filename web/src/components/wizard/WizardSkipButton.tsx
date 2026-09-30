import { useNavigate } from "react-router";

import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Leaves the wizard for the board, or home when the viewer can't read tickets: the project step already created the project, so skipping keeps it.
export const WizardSkipButton = () => {
  const navigate = useNavigate();
const wsPath = useWorkspacePath();
  const projectId = useProjectWizardStore((s) => s.projectId);
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;

  const skip = () => {
    if (!canOpenBoard) return void navigate(wsPath("/"));
    void navigate(wsPath(projectId ? `/board/${projectId}` : "/board"));
  };

  return <WizardSkipLink onClick={skip} />;
};
