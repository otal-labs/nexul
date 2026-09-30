import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

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
  stackId: string;
  active: StackSection;
  // A branch deployment has no rules of its own — the nav must not link to an empty section.
  showBranches: boolean;
  // Exposures are DNS, read with dns:read rather than the stack's own bit.
  showExposures: boolean;
}

const visible = (section: StackSection, showBranches: boolean, showExposures: boolean) =>
  (section !== "branches" || showBranches) && (section !== "exposures" || showExposures);

export const StackNav = ({ stackId, active, showBranches, showExposures }: StackNavProps) => {
  const wsPath = useWorkspacePath();
  return (
    <SettingsSectionNav
      ariaLabel="Stack sections"
      basePath={wsPath(`/stacks/${stackId}`)}
      active={active}
      items={STACK_SECTIONS.filter((section) => visible(section, showBranches, showExposures)).map((section) => ({
        section,
        label: sectionLabels[section],
        danger: section === "danger",
      }))}
    />
  );
};
