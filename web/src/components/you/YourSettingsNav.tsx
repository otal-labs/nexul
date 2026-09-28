import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";

export const YOUR_SETTINGS_SECTIONS = ["profile", "appearance", "security", "pairing"] as const;

export type YourSettingsSection = (typeof YOUR_SETTINGS_SECTIONS)[number];

export const DEFAULT_YOUR_SETTINGS_SECTION: YourSettingsSection = "profile";

export const isYourSettingsSection = (value: string | null | undefined): value is YourSettingsSection =>
  !!value && (YOUR_SETTINGS_SECTIONS as readonly string[]).includes(value);

const sectionLabels: Record<YourSettingsSection, string> = {
  profile: "Profile",
  appearance: "Appearance",
  security: "Security",
  pairing: "T3 pairing",
};

interface YourSettingsNavProps {
  active: YourSettingsSection;
}

export const YourSettingsNav = ({ active }: YourSettingsNavProps) => (
  <SettingsSectionNav
    ariaLabel="Your settings sections"
    basePath="/settings"
    active={active}
    items={YOUR_SETTINGS_SECTIONS.map((section) => ({ section, label: sectionLabels[section] }))}
  />
);
