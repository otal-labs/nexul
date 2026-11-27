import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { providerSetupLines, type Computer, type ComputerSetup } from "@/models/Pairing";

const modelName = (computer: Computer, setup: ComputerSetup, driver: string) => {
  const slug = setup.models[driver] ?? "";
  if (slug === "") return "Default";
  const models = computer.facts?.providers.find((p) => p.driver.toLowerCase() === driver.toLowerCase())?.models ?? [];
  return models.find((m) => m.slug === slug)?.name ?? slug;
};

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
          <li key={line.provider} className="flex items-center gap-1.5" title={line.name}>
            <HarnessProviderMark driver={line.provider} />
            <span className="sr-only">{line.name}</span>
            <span className="font-mono text-xs">{modelName(computer, setup, line.provider)}</span>
          </li>
        ))}
      </ul>
    </SettingsRow>
  );
};
