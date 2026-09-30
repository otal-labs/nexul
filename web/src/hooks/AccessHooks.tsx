import { isInstanceSection, type SettingsSection, visibleSettingsSections } from "@/components/settings/SettingsNav";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { AREA_PERMISSION, INSTANCE_SECTION_PERMISSION, type Area, type InstanceSection, type RouteArea } from "@/models/Access";
import { hasPermission } from "@/models/Permission";

// Undefined until permissions first arrive; isFetched, since a failed read refetches as pending and must not blink.
export const useAreaAccess = (): ((area: Area) => boolean) | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data, isFetched } = useFetchMyRole(selectedWorkspaceId);
  if (!isFetched) return undefined;
  return (area) => hasPermission(data?.permissions, AREA_PERMISSION[area]);
};

// Whether the viewer holds value in any workspace, which is what the server checks an instance-level action against.
export const useHasInstancePermission = (value: string): boolean => {
  const { data: me } = useFetchMe();
  return hasPermission(me?.instance_permissions, value);
};

// Every Configuration and Settings section the viewer may open, in nav order; undefined until every gate behind them has answered.
export const useVisibleSettingsSections = (): SettingsSection[] | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: me } = useFetchMe();
  const { data: role, isFetched: roleFetched } = useFetchMyRole(selectedWorkspaceId);
  const permissions = role?.permissions ?? [];
  const anywhere = me?.instance_permissions;
  const instanceSections = (Object.keys(INSTANCE_SECTION_PERMISSION) as InstanceSection[]).filter((section) =>
    hasPermission(anywhere, INSTANCE_SECTION_PERMISSION[section]),
  );

  if (!me || !roleFetched) return undefined;
  return visibleSettingsSections({
    instanceSections,
    teamIsInstanceWide: hasPermission(anywhere, "accounts:read"),
    showTeam: hasPermission(anywhere, "accounts:read") || hasPermission(anywhere, "members:write"),
    showRoles: hasPermission(permissions, "roles:write"),
    showPlays: hasPermission(permissions, "plays:read"),
    showInterviewTemplate: hasPermission(permissions, "memories:read"),
    showMentionLayout: hasPermission(permissions, "workspaces:write"),
  });
};

// The workspace sections of Configuration the viewer may open; the instance ones live on the Settings page.
export const useConfigurationSections = (): SettingsSection[] | undefined => {
  const sections = useVisibleSettingsSections();
  const teamIsInstanceWide = useHasInstancePermission("accounts:read");
  return sections?.filter((section) => !isInstanceSection(section, teamIsInstanceWide));
};

// The instance sections the viewer may open from the Settings page.
export const useInstanceSettingsSections = (): SettingsSection[] | undefined => {
  const sections = useVisibleSettingsSections();
  const teamIsInstanceWide = useHasInstancePermission("accounts:read");
  return sections?.filter((section) => isInstanceSection(section, teamIsInstanceWide));
};

// Whether one settings section is open to the viewer; false while that is still loading, so a link never flashes.
export const useCanOpenSection = (section: SettingsSection): boolean =>
  useVisibleSettingsSections()?.includes(section) ?? false;

// Danger zone holds no action yet, so on its own it doesn't make Configuration worth opening.
const opensConfiguration = (sections: SettingsSection[]) => sections.some((section) => section !== "danger");

// Whether the viewer may open an area's page; undefined while that is still loading.
export const useCanOpen = (): ((area: RouteArea) => boolean | undefined) => {
  const can = useAreaAccess();
  const sections = useConfigurationSections();
  return (area) => {
    if (area === "configuration") return sections && opensConfiguration(sections);
    return can && can(area);
  };
};
