import { useNavigate } from "react-router";

import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Leaves the wizard for the board, or home when the viewer can't read tickets: the project step already created the project, so skipping keeps it.
export const WizardSkipButton = () => {
  const navigate = useNavigate();
  const projectId = useProjectWizardStore((s) => s.projectId);
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;

  const skip = () => {
    if (!canOpenBoard) return void navigate("/");
    void navigate(projectId ? `/board/${projectId}` : "/board");
  };

  return <WizardSkipLink onClick={skip} />;
};
