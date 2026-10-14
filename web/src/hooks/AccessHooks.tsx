import { AREA_PERMISSION, hasPermission, projectPermissions, workspaceWidePermissions, type Area } from "@nexul/client-core/permissions";

import { isInstanceSection, type SettingsSection, visibleSettingsSections } from "@/components/settings/SettingsNav";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useFetchMachines } from "@/hooks/RunnerHooks";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import { canOpenWizard, INSTANCE_SECTION_PERMISSION, type InstanceSection, type RouteArea } from "@/models/Access";
import type { WorkspaceAccess } from "@/models/Workspace";

// Undefined until permissions first arrive; isFetched, since a failed read refetches as pending and must not blink.
// A project area answers for projectId, else the project in view, since a member holds it only in projects they're in.
export const useAreaAccess = (projectId?: string): ((area: Area) => boolean) | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const selectedProjectId = useWorkspaceStore((s) => s.selectedProjectId);
  const { data, isFetched } = useFetchMyRole(selectedWorkspaceId);
  if (!isFetched) return undefined;
  const held = projectPermissions(data, projectId ?? selectedProjectId);
  return (area) => hasPermission(held, AREA_PERMISSION[area]);
};

// Whether the viewer holds an area in any project, for what stands for all of them (the sidebar's project section).
export const useAnyProjectAreaAccess = (): ((area: Area) => boolean) | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data, isFetched } = useFetchMyRole(selectedWorkspaceId);
  if (!isFetched) return undefined;
  const held = workspaceWidePermissions(data);
  return (area) => hasPermission(held, AREA_PERMISSION[area]);
};

// Manage this project's access, held through a role or on one project: what opens the Access page.
const MANAGES_ACCESS = "projects:access";

// Whether the viewer holds value through a role or in any project they are in; undefined while that is still loading.
const useAnyProjectAreaAccessOf = (value: string): boolean | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data, isFetched } = useFetchMyRole(selectedWorkspaceId);
  if (!isFetched) return undefined;
  return hasPermission(workspaceWidePermissions(data), value);
};

// Whether the viewer holds value in any workspace, which is what the server checks an instance-level action against.
export const useHasInstancePermission = (value: string): boolean => {
  const { data: me } = useFetchMe();
  return hasPermission(me?.instance_permissions, value);
};

// Whether the viewer is the instance Owner, who alone sees and sets the rules on instance areas.
export const useIsInstanceOwner = (): boolean => !!useFetchMe().data?.instance_owner;

// The Runners link is for whoever sees a machine or can add one; the route itself still opens for every member.
export const useShowRunnersLink = (): boolean => {
  const { data: me } = useFetchMe();
  const { data: machines } = useFetchMachines();
  return (machines?.length ?? 0) > 0 || me?.instance_owner === true || hasPermission(me?.instance_permissions, "machines:create");
};

// Every Configuration and Settings section open to someone holding anywhere in some workspace and permissions in this one.
const settingsSectionsFor = (anywhere: string[] | undefined, permissions: string[]): SettingsSection[] => {
  const instanceSections = (Object.keys(INSTANCE_SECTION_PERMISSION) as InstanceSection[]).filter((section) =>
    hasPermission(anywhere, INSTANCE_SECTION_PERMISSION[section]),
  );
  return visibleSettingsSections({
    instanceSections,
    teamIsInstanceWide: hasPermission(anywhere, "accounts:read"),
    showTeam: hasPermission(anywhere, "accounts:read") || hasPermission(anywhere, "members:write"),
    showGeneral: hasPermission(permissions, "workspaces:write"),
    showRoles: hasPermission(permissions, "roles:write"),
    showPlays: hasPermission(permissions, "plays:read"),
    showInterviewTemplate: hasPermission(permissions, "memories:read") || hasPermission(permissions, "memories:template"),
    showMentionLayout: hasPermission(permissions, "workspaces:write"),
  });
};

// Danger zone holds no action yet, so on its own it doesn't make Configuration worth opening.
const opensConfiguration = (sections: SettingsSection[]) => sections.some((section) => section !== "danger");

// What a viewer may open in another workspace, for the switcher to land on a page that exists for them there.
export const workspaceAccess = (anywhere: string[] | undefined, permissions: string[]): WorkspaceAccess => {
  const teamIsInstanceWide = hasPermission(anywhere, "accounts:read");
  const configurationSections = settingsSectionsFor(anywhere, permissions).filter(
    (section) => !isInstanceSection(section, teamIsInstanceWide),
  );
  return {
    configurationSections,
    canOpen: (area) => {
      if (area === "configuration") return opensConfiguration(configurationSections);
      if (area === "access") return hasPermission(permissions, MANAGES_ACCESS);
      if (area === "wizard") return canOpenWizard((held) => hasPermission(permissions, AREA_PERMISSION[held]));
      return hasPermission(permissions, AREA_PERMISSION[area]);
    },
  };
};

// Every Configuration and Settings section the viewer may open, in nav order; undefined until every gate behind them has answered.
export const useVisibleSettingsSections = (): SettingsSection[] | undefined => {
  const selectedWorkspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: me } = useFetchMe();
  const { data: role, isFetched: roleFetched } = useFetchMyRole(selectedWorkspaceId);

  if (!me || !roleFetched) return undefined;
  return settingsSectionsFor(me.instance_permissions, role?.permissions ?? []);
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

// Whether the viewer may open an area's page; undefined while that is still loading.
export const useCanOpen = (): ((area: RouteArea) => boolean | undefined) => {
  const can = useAreaAccess();
  const anywhere = useAnyProjectAreaAccessOf(MANAGES_ACCESS);
  const sections = useConfigurationSections();
  return (area) => {
    if (area === "configuration") return sections && opensConfiguration(sections);
    if (area === "access") return anywhere;
    if (area === "wizard") return can && canOpenWizard(can);
    return can && can(area);
  };
};
