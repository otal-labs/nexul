import { useQueryClient } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { workspaceAccess } from "@/hooks/AccessHooks";
import { useFetchMe } from "@/hooks/AuthHooks";
import { myRoleQuery, useFetchWorkspaces } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { workspaceWidePermissions } from "@/models/Permission";
import { switchWorkspacePath, workspacePath } from "@/models/Workspace";

// The workspaces and the one in view, and a switch that keeps the reader on the same section in the other one.
export const useSwitchWorkspace = () => {
  // Server state, rendered straight from the query cache, never mirrored into workspaceStore (F5).
  const { data: workspaces } = useFetchWorkspaces();
  const { selectedWorkspaceId, selectWorkspace } = useWorkspaceStore(
    useShallow((s) => ({ selectedWorkspaceId: s.selectedWorkspaceId, selectWorkspace: s.selectWorkspace })),
  );
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const client = useQueryClient();
  const { data: me } = useFetchMe();
  const current = workspaces?.find((w) => w.id === selectedWorkspaceId) ?? workspaces?.[0];

  // Inside a workspace the URL decides: the same section in the other one. A personal page has no workspace URL to change.
  const switchTo = async (workspaceId: string) => {
    const target = workspaces?.find((w) => w.id === workspaceId);
    const prefix = current ? workspacePath(current.slug, "/") : "";
    const inWorkspace = !!current && (pathname === prefix || pathname.startsWith(`${prefix}/`));
    if (!target || !inWorkspace) return selectWorkspace(workspaceId, target?.slug ?? "");
    const role = await client.fetchQuery(myRoleQuery(workspaceId)).catch(() => undefined);
    void navigate(switchWorkspacePath(pathname, target.slug, workspaceAccess(me?.instance_permissions, workspaceWidePermissions(role))));
  };

  return { workspaces, current, selectedWorkspaceId, switchTo };
};
