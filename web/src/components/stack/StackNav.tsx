import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";

export const STACK_SECTIONS = ["overview", "exposures", "branches", "history", "danger"] as const;

export type StackSection = (typeof STACK_SECTIONS)[number];

export const DEFAULT_STACK_SECTION: StackSection = "overview";

export const isStackSection = (value: string | null | undefined): value is StackSection =>
  !!value && (STACK_SECTIONS as readonly string[]).includes(value);

const sectionLabels: Record<StackSection, string> = {
  overview: "Overview",
  exposures: "Exposures",
  branches: "Branch deploys",
  history: "Deploy history",
  danger: "Danger zone",
};

interface StackNavProps {
  active: StackSection;
  // A branch deployment has no rules of its own — the nav must not link to an empty section.
  showBranches: boolean;
}

export const StackNav = ({ active, showBranches }: StackNavProps) => (
  <SettingsSectionNav
    ariaLabel="Stack sections"
    active={active}
    items={STACK_SECTIONS.filter((section) => section !== "branches" || showBranches).map((section) => ({
      section,
      label: sectionLabels[section],
      danger: section === "danger",
    }))}
  />
);
