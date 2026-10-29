import { SettingsSectionNav } from "@/components/settings/SettingsSectionNav";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";

// The stack's configuration, behind its Settings tab; the daily views (Overview, Deploys, Logs) are tabs of their own.
export const STACK_SECTIONS = ["environment", "alerts", "exposures", "branches", "danger"] as const;

export type StackSection = (typeof STACK_SECTIONS)[number];

export const DEFAULT_STACK_SECTION: StackSection = "environment";

export const isStackSection = (value: string | null | undefined): value is StackSection =>
  !!value && (STACK_SECTIONS as readonly string[]).includes(value);

const sectionLabels: Record<StackSection, string> = {
  environment: "Environment",
  alerts: "Alerts",
  exposures: "Exposures",
  branches: "Branch deploys",
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
      ariaLabel="Stack settings"
      basePath={wsPath(`/stacks/${stackId}/settings`)}
      active={active}
      items={STACK_SECTIONS.filter((section) => visible(section, showBranches, showExposures)).map((section) => ({
        section,
        label: sectionLabels[section],
        danger: section === "danger",
      }))}
    />
  );
};
