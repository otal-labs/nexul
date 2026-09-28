import { useNavigate } from "react-router";

import { Button } from "@/components/ui/button";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Leaves the wizard for the board: the project step already created the project, so skipping keeps it.
export const WizardSkipButton = () => {
  const navigate = useNavigate();
  const projectId = useProjectWizardStore((s) => s.projectId);

  return (
    <Button
      type="button"
      variant="ghost"
      className="text-muted-foreground"
      onClick={() => void navigate(projectId ? `/board/${projectId}` : "/board")}
    >
      Skip for now
    </Button>
  );
};
