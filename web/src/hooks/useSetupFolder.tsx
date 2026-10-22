import { useCallback } from "react";

import { useFetchHarnessProjects, useListComputers } from "@/hooks/PairingHooks";
import { useSetupDraftStore } from "@/stores/setupDraftStore";

// The Set up step's folder: the unsaved pick, else the saved one, else the one setup would fall back to (the computer's chat
// folder, else T3's first project).
export const useSetupFolder = (computerId: string, saved: string) => {
  const { data: projects } = useFetchHarnessProjects(computerId);
  const { data: computers } = useListComputers();
  const picked = useSetupDraftStore((s) => s.drafts[computerId]?.folder);
  const edit = useSetupDraftStore((s) => s.edit);
  const fallback = computers?.find((c) => c.id === computerId)?.chat_project_id;
  // A saved folder T3 Code no longer opens falls back like no pick at all.
  const stillOpen = projects?.some((p) => p.path === saved) ? saved : "";
  const preselected = (projects?.find((p) => p.id === fallback) ?? projects?.[0])?.path ?? "";
  const pick = useCallback((folder: string) => edit(computerId, (d) => ({ ...d, folder })), [computerId, edit]);
  return { projects: projects ?? [], folder: picked || stillOpen || preselected, pick };
};
