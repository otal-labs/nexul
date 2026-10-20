import { useParams, useSearchParams } from "react-router";

import { useWizardEnvKeys } from "@/hooks/useWizardSetup";
import { wizardOrder, type WizardDoor, type WizardStepId } from "@/models/ProjectWizard";

// The URL picks the door: ?stack= attaches a repository to a stack, ?add= adds a service, else the project's setup.
export const useWizardDoor = (): WizardDoor => {
  const [searchParams] = useSearchParams();
  if (searchParams.has("stack")) return "attach";
  return searchParams.has("add") ? "add" : "setup";
};

export const useWizardStepOrder = (): WizardStepId[] => {
  const door = useWizardDoor();
  const { step } = useParams();
  const showEnv = useWizardEnvKeys().length > 0 || step === "env";
  return wizardOrder(door, showEnv);
};
