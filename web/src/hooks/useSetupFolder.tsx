import { useState } from "react";

import { useFetchHarnessProjects, useFetchPairingDefaults } from "@/hooks/PairingHooks";

// The Set up step's folder: the user's pick, else the project setup would fall back to (the default project, else T3's first).
export const useSetupFolder = (computerId: string) => {
  const { data: projects } = useFetchHarnessProjects(computerId);
  const { data: defaults } = useFetchPairingDefaults();
  const [picked, setPicked] = useState("");
  const fallback = defaults?.default_computer_id === computerId ? defaults.fallback_project_id : undefined;
  const preselected = (projects?.find((p) => p.id === fallback) ?? projects?.[0])?.path ?? "";
  return { projects: projects ?? [], folder: picked || preselected, pick: setPicked };
};
