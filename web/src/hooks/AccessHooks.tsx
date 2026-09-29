import { type SettingsSection, visibleSettingsSections } from "@/components/settings/SettingsNav";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchTeam } from "@/hooks/TeamHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { AREA_PERMISSION, type Area, type RouteArea } from "@/models/Access";
import { hasPermission } from "@/models/Permission";

// Undefined until permissions first arrive; isFetched, since a failed read refetches as pending and must not blink.
export const useAreaAccess = (): ((area: Area) => boolean) | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data, isFetched } = useFetchMyRole(selectedWorkspaceId);
  if (!isFetched) return undefined;
  return (area) => hasPermission(data?.permissions, AREA_PERMISSION[area]);
};

// The Configuration sections the viewer may open, in nav order; undefined until every gate behind them has answered.
export const useVisibleSettingsSections = (): SettingsSection[] | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: me } = useFetchMe();
  const { data: role, isFetched: roleFetched } = useFetchMyRole(selectedWorkspaceId);
  const permissions = role?.permissions ?? [];
  const isInstanceAdmin = me?.user?.can_create_workspace ?? false;
  const canManageMembers = hasPermission(permissions, "members:write");
  // Members may be managed in a workspace other than the selected one; the scoped Team read answers that.
  const needsTeam = !!me && !isInstanceAdmin && !canManageMembers && roleFetched;
  const { data: team, isFetched: teamFetched } = useFetchTeam(needsTeam);

  if (!me || !roleFetched || (needsTeam && !teamFetched)) return undefined;
  return visibleSettingsSections({
    isInstanceAdmin,
    showRoles: hasPermission(permissions, "roles:write"),
    showPlays: hasPermission(permissions, "plays:read"),
    showInterviewTemplate: hasPermission(permissions, "memories:read"),
    showMentionLayout: hasPermission(permissions, "workspaces:write"),
    showTeam: canManageMembers || !!team,
  });
};

// Danger zone holds no action yet, so on its own it doesn't make Configuration worth opening.
const opensConfiguration = (sections: SettingsSection[]) => sections.some((section) => section !== "danger");

// Whether the viewer may open an area's page; undefined while that is still loading.
export const useCanOpen = (): ((area: RouteArea) => boolean | undefined) => {
  const can = useAreaAccess();
  const sections = useVisibleSettingsSections();
  return (area) => {
    if (area === "configuration") return sections && opensConfiguration(sections);
    return can && can(area);
  };
};
