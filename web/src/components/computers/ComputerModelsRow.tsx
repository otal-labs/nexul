import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { computerModelName } from "@/models/ModelPick";
import { providerSetupLines, type Computer, type ComputerSetup } from "@/models/Pairing";

interface ComputerModelsRowProps {
  computer: Computer;
  setup: ComputerSetup;
}

// The model each confirmed provider runs on here, by its mark; picked in the Set up step.
export const ComputerModelsRow = ({ computer, setup }: ComputerModelsRowProps) => {
  const lines = providerSetupLines(setup).filter((l) => l.state === "confirmed");
  if (lines.length === 0) return null;

  return (
    <SettingsRow label="Models" stacked className="last:pb-4">
      <ul className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
        {lines.map((line) => (
          <li key={line.provider} className="flex items-center gap-1.5" title={`${line.name} · ${computerModelName(computer, setup, line.provider)}`}>
            <HarnessProviderMark driver={line.provider} />
            <span className="sr-only">{line.name}</span>
            <span className="font-mono text-xs">{computerModelName(computer, setup, line.provider)}</span>
          </li>
        ))}
      </ul>
    </SettingsRow>
  );
};
