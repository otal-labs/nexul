import { useCallback } from "react";

import { useFetchHarnessProjects, useFetchPairingDefaults } from "@/hooks/PairingHooks";
import { useSetupDraftStore } from "@/stores/setupDraftStore";

// The Set up step's folder: the unsaved pick, else the saved one, else the project setup would fall back to (the default project, else T3's first).
export const useSetupFolder = (computerId: string, saved: string) => {
  const { data: projects } = useFetchHarnessProjects(computerId);
  const { data: defaults } = useFetchPairingDefaults();
  const picked = useSetupDraftStore((s) => s.drafts[computerId]?.folder);
  const edit = useSetupDraftStore((s) => s.edit);
  const fallback = defaults?.default_computer_id === computerId ? defaults.fallback_project_id : undefined;
  // A saved folder T3 Code no longer opens falls back like no pick at all.
  const stillOpen = projects?.some((p) => p.path === saved) ? saved : "";
  const preselected = (projects?.find((p) => p.id === fallback) ?? projects?.[0])?.path ?? "";
  const pick = useCallback((folder: string) => edit(computerId, (d) => ({ ...d, folder })), [computerId, edit]);
  return { projects: projects ?? [], folder: picked || stillOpen || preselected, pick };
};
