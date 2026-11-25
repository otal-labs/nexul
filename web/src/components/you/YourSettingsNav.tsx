import { SettingsSectionNav, type SettingsSectionNavItem } from "@/components/settings/SettingsSectionNav";
import { INSTANCE_GROUP, sectionLabels as instanceLabels, type SettingsSection } from "@/components/settings/SettingsNav";
import { useInstanceSettingsSections } from "@/hooks/AccessHooks";
import { useSkillsOutdated } from "@/hooks/ComputerSetupHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { AGENT_PATH } from "@/models/Agent";
import { SKILLS_OUTDATED } from "@/models/Pairing";

export const YOUR_SETTINGS_SECTIONS = ["profile", "appearance", "security"] as const;

export type YourSettingsSection = (typeof YOUR_SETTINGS_SECTIONS)[number];

export const DEFAULT_YOUR_SETTINGS_SECTION: YourSettingsSection = "profile";

export const isYourSettingsSection = (value: string | null | undefined): value is YourSettingsSection =>
  !!value && (YOUR_SETTINGS_SECTIONS as readonly string[]).includes(value);

const YOUR_GROUP = "You";

export const sectionLabels: Record<YourSettingsSection, string> = {
  profile: "Profile",
  appearance: "Appearance",
  security: "Security",
};

interface YourSettingsNavProps {
  active: YourSettingsSection | SettingsSection;
}

// The instance group appears only for a viewer holding one of its permissions; without it the personal list carries no label.
export const YourSettingsNav = ({ active }: YourSettingsNavProps) => {
  const instanceSections = useInstanceSettingsSections() ?? [];
  const skillsOutdated = useSkillsOutdated();
  const personalGroup = instanceSections.length > 0 ? YOUR_GROUP : undefined;
  const wsPath = useWorkspacePath();
  const items: SettingsSectionNavItem[] = [
    ...YOUR_SETTINGS_SECTIONS.map((section) => ({ section, label: sectionLabels[section], group: personalGroup })),
    // Your computers live on the workspace's Agent page (ADR 0211); the link keeps them findable from here.
    { section: "computers", label: "Computers", group: personalGroup, to: wsPath(AGENT_PATH), dot: skillsOutdated ? SKILLS_OUTDATED : undefined },
    ...instanceSections.map((section) => ({ section, label: instanceLabels[section], group: INSTANCE_GROUP })),
  ];

  return <SettingsSectionNav ariaLabel="Settings sections" basePath="/settings" active={active} items={items} />;
};
