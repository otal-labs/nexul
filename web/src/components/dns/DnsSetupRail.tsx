import { WizardRail } from "@/components/wizard/WizardRail";
import { WizardRailRow } from "@/components/wizard/WizardRailRow";
import type { DnsSetup } from "@/hooks/useDnsSetup";
import type { WizardRailMark } from "@/models/WizardRail";

interface DnsSetupRailProps {
  setup: DnsSetup;
}

// Any rung its groundwork has unlocked opens from here; opening one never changes anything, only its Change does.
export const DnsSetupRail = ({ setup }: DnsSetupRailProps) => {
  const { rungs, view, open } = setup;
  const markOf = (key: string, state: string): WizardRailMark => {
    if (key === view) return "current";
    if (state === "done") return "done";
    if (state === "locked") return "locked";
    return "left";
  };
  return (
    <WizardRail ariaLabel="DNS setup steps" label="DNS" done={rungs.filter((rung) => rung.state === "done").length} total={rungs.length}>
      {rungs.map((rung, i) => (
        <WizardRailRow
          key={rung.key}
          label={rung.label}
          mark={markOf(rung.key, rung.state)}
          ordinal={String(i + 1).padStart(2, "0")}
          total={rungs.length}
          onOpen={rung.state === "locked" ? undefined : () => open(rung.key)}
          last={i === rungs.length - 1}
        />
      ))}
    </WizardRail>
  );
};
