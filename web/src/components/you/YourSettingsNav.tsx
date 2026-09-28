import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";

export const YOUR_SETTINGS_SECTIONS = ["profile", "appearance", "security", "pairing"] as const;

export type YourSettingsSection = (typeof YOUR_SETTINGS_SECTIONS)[number];

export const isYourSettingsSection = (value: string | null): value is YourSettingsSection =>
  !!value && (YOUR_SETTINGS_SECTIONS as readonly string[]).includes(value);

const labels: Record<YourSettingsSection, string> = {
  profile: "Profile",
  appearance: "Appearance",
  security: "Security",
  pairing: "T3 pairing",
};

export const YourSettingsNav = ({ active }: { active: YourSettingsSection }) => (
  <SettingsSectionNav
    ariaLabel="Your settings sections"
    active={active}
    items={YOUR_SETTINGS_SECTIONS.map((section) => ({ section, label: labels[section] }))}
  />
);
