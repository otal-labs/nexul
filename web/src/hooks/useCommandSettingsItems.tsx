import { ServerIcon, SettingsIcon, ShieldCheck, SlidersHorizontalIcon, UserCogIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { PROJECT_SETTINGS_SECTIONS, sectionLabels as projectLabels, type ProjectSettingsSection } from "@/components/settings/ProjectSettingsNav";
import { sectionLabels, type SettingsSection } from "@/components/settings/SettingsNav";
import { YOUR_SETTINGS_SECTIONS, sectionLabels as yourLabels, type YourSettingsSection } from "@/components/you/YourSettingsNav";
import { useCanOpen, useConfigurationSections, useInstanceSettingsSections, useManagesAccess, useAreaAccess } from "@/hooks/AccessHooks";
import { useIsWorkspaceOwner } from "@/hooks/AccessGridHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import type { CommandGroup, CommandItem } from "@/models/Command";
import { projectSettingsPath, projectToken } from "@/models/Project";

interface Row {
  label: string;
  path: string;
  hint: string;
  keywords?: string | undefined;
}

// An empty value is the page's first tab, which PageTabs serves at the bare section path.
type Tab = [value: string, label: string, keywords?: string];

const tabRows = (base: string, hint: string, tabs: Tab[]): Row[] =>
  tabs.map(([value, label, keywords]) => ({ label, path: value ? `${base}/${value}` : base, hint, keywords }));

const YOUR_KEYWORDS: Partial<Record<YourSettingsSection, string>> = {
  appearance: "theme dark light mode palette colours",
  security: "devices phone sessions sign out tokens api mcp personal access key",
  pairing: "pair t3 computer",
};

// Computers' first tab is the section row itself, so only its other tabs are listed.
const YOUR_TABS: Partial<Record<YourSettingsSection, Tab[]>> = {
  pairing: [
    ["projects", "Projects"],
    ["defaults", "Defaults"],
  ],
};

const SECTION_KEYWORDS: Partial<Record<SettingsSection, string>> = {
  team: "people members users accounts",
  "sign-in": "login oauth",
  connectors: "github cloudflare integrations",
  dns: "domains records tunnels",
  templates: "agent prompt intro footer",
  instance: "upgrade version",
  interview: "questions",
  mentions: "mentions",
};

const SECTION_TABS: Partial<Record<SettingsSection, Tab[]>> = {
  team: [["invitations", "Invitations", "invite link people"]],
  "sign-in": [
    ["", "Discord"],
    ["google", "Google"],
  ],
};

const PROJECT_KEYWORDS: Partial<Record<ProjectSettingsSection, string>> = {
  services: "stacks deploys",
  repositories: "github repo",
  categories: "swimlanes",
  danger: "delete archive",
};

const BOARD_TABS: Tab[] = [
  ["", "Status columns", "statuses"],
  ["types", "Ticket types"],
  ["labels", "Label colors", "labels tags"],
];

const yourRows = (): Row[] =>
  YOUR_SETTINGS_SECTIONS.flatMap((section) => [
    { label: yourLabels[section], path: `/settings/${section}`, hint: "Settings", keywords: YOUR_KEYWORDS[section] },
    ...tabRows(`/settings/${section}`, `Settings › ${yourLabels[section]}`, YOUR_TABS[section] ?? []),
  ]);

const sectionRows = (sections: SettingsSection[], base: string, hint: string): Row[] =>
  sections.flatMap((section) => [
    { label: sectionLabels[section], path: `${base}/${section}`, hint, keywords: SECTION_KEYWORDS[section] },
    ...tabRows(`${base}/${section}`, `${hint} › ${sectionLabels[section]}`, SECTION_TABS[section] ?? []),
  ]);

const projectRows = (base: string, name: string): Row[] =>
  PROJECT_SETTINGS_SECTIONS.flatMap((section) => {
    const path = section === "general" ? base : `${base}/${section}`;
    return [
      { label: projectLabels[section], path, hint: `${name} settings`, keywords: PROJECT_KEYWORDS[section] },
      ...(section === "board" ? tabRows(path, `${name} settings › Board`, BOARD_TABS) : []),
    ];
  });

// Settings sections and tabs the viewer can open, found by typing; empty, the palette lists none.
export const useCommandSettingsItems = (browsing: boolean): CommandGroup[] => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { current } = useSidebarProject();
  const canOpen = useCanOpen();
  const instanceSections = useInstanceSettingsSections();
  const configurationSections = useConfigurationSections();
  const managesAccess = useManagesAccess();
  const canWriteRoles = useHasPermission("roles:write");
  const owner = useIsWorkspaceOwner();
  const can = useAreaAccess(current?.id);

  const items = (icon: CommandItem["icon"], rows: Row[], inWorkspace: boolean): CommandItem[] =>
    rows.map(({ label, path, hint, keywords }) => ({
      id: `setting:${hint}:${label}`,
      label,
      icon,
      hint,
      ...(keywords && { keywords }),
      run: () => void navigate(inWorkspace ? wsPath(path) : path),
    }));

  if (browsing) return [];

  const access: Row[] = [
    ...(managesAccess ? [{ label: "Project access", path: "/access", hint: "Access", keywords: "who can see projects permissions" }] : []),
    ...(canWriteRoles ? [{ label: "Roles", path: "/access/roles", hint: "Access", keywords: "permissions role" }] : []),
    ...(owner ? [{ label: "Public channels", path: "/access/channels", hint: "Access" }] : []),
  ];
  const project = current && can?.("projects") ? projectRows(projectSettingsPath(projectToken(current)), current.name) : [];

  return [
    {
      heading: "Settings",
      items: [
        ...items(UserCogIcon, yourRows(), false),
        ...items(ServerIcon, sectionRows(instanceSections ?? [], "/settings", "Settings"), false),
        ...items(SlidersHorizontalIcon, sectionRows(configurationSections ?? [], "/configuration", "Configuration"), true),
        ...(canOpen("access") === true ? items(ShieldCheck, access, true) : []),
        ...items(SettingsIcon, project, true),
      ],
    },
  ];
};
