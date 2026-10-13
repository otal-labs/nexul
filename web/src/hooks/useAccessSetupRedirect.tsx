import { useEffect } from "react";
import { useNavigate } from "react-router";

import { useAccessSetupState } from "@/hooks/AccessGridHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useAccessSetupStore } from "@/stores/accessSetupStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// An Owner's first visit after the upgrade goes once to the access setup; the sidebar's Set up access gets them back.
export const useAccessSetupRedirect = () => {
  const settingUp = useAccessSetupState() === "owner";
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const wsPath = useWorkspacePath();
  const navigate = useNavigate();
  useEffect(() => {
    const store = useAccessSetupStore.getState();
    if (!settingUp || store.redirected.includes(workspaceId)) return;
    store.markRedirected(workspaceId);
    void navigate(wsPath("/access"), { replace: true });
  }, [settingUp, workspaceId, navigate, wsPath]);
};
