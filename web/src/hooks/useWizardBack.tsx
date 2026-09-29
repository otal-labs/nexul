import { useLocation, useNavigate, useSearchParams } from "react-router";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Only rungs with nothing to undo get a way back: Info leaves (nothing created yet), Service returns to Repository
// until its stack exists. Repository would land on Info and make a second project; later rungs sit after a created stack.
export const useWizardBack = (step: string | undefined): (() => void) | undefined => {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const canOpenBoard = useAreaAccess()?.("tickets") ?? false;
  const stackId = useProjectWizardStore((s) => s.stackId);

  const leave = () => {
    // "default" is the first entry of the session, so there is no in-app page to return to.
    if (location.key !== "default") return void navigate(-1);
    void navigate(canOpenBoard ? "/board" : "/");
  };

  const toRepository = () => {
    const query = searchParams.toString();
    void navigate(`/wizard/project/repository${query ? `?${query}` : ""}`);
  };

  if (step === "project") return leave;
  if (step === "service" && !stackId) return toRepository;
  return undefined;
};
