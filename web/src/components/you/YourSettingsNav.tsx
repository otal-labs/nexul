import { SettingsSectionNav, type SettingsSectionNavItem } from "@/components/settings/SettingsSectionNav";
import { INSTANCE_GROUP, sectionLabels as instanceLabels, type SettingsSection } from "@/components/settings/SettingsNav";
import { useInstanceSettingsSections } from "@/hooks/AccessHooks";
import { useSkillsOutdated } from "@/hooks/ComputerSetupHooks";
import { SKILLS_OUTDATED } from "@/models/Pairing";

export const YOUR_SETTINGS_SECTIONS = ["profile", "appearance", "security", "pairing"] as const;

export type YourSettingsSection = (typeof YOUR_SETTINGS_SECTIONS)[number];

export const DEFAULT_YOUR_SETTINGS_SECTION: YourSettingsSection = "profile";

export const isYourSettingsSection = (value: string | null | undefined): value is YourSettingsSection =>
  !!value && (YOUR_SETTINGS_SECTIONS as readonly string[]).includes(value);

const YOUR_GROUP = "You";

const sectionLabels: Record<YourSettingsSection, string> = {
  profile: "Profile",
  appearance: "Appearance",
  security: "Security",
  pairing: "T3 pairing",
};

interface YourSettingsNavProps {
  active: YourSettingsSection | SettingsSection;
}

// The instance group appears only for a viewer holding one of its permissions; without it the personal list carries no label.
export const YourSettingsNav = ({ active }: YourSettingsNavProps) => {
  const instanceSections = useInstanceSettingsSections() ?? [];
  const skillsOutdated = useSkillsOutdated();
  const personalGroup = instanceSections.length > 0 ? YOUR_GROUP : undefined;
  const dots: Partial<Record<YourSettingsSection, string>> = skillsOutdated ? { pairing: SKILLS_OUTDATED } : {};
  const items: SettingsSectionNavItem[] = [
    ...YOUR_SETTINGS_SECTIONS.map((section) => ({ section, label: sectionLabels[section], group: personalGroup, dot: dots[section] })),
    ...instanceSections.map((section) => ({ section, label: instanceLabels[section], group: INSTANCE_GROUP })),
  ];

  return <SettingsSectionNav ariaLabel="Settings sections" basePath="/settings" active={active} items={items} />;
};
