import { ComputerChatFolderRow } from "@/components/computers/ComputerChatFolderRow";
import { ComputerModelsRow } from "@/components/computers/ComputerModelsRow";
import { ComputerRunsAtOnceRow } from "@/components/computers/ComputerRunsAtOnceRow";
import { ComputerSwitchRow } from "@/components/computers/ComputerSwitchRow";
import { ComputerLabels } from "@/components/settings/ComputerLabels";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { useUpdateComputerSettings } from "@/hooks/PairingHooks";
import type { Computer, ComputerSetup } from "@/models/Pairing";

interface ComputerSettingsCardProps {
  computer: Computer;
  setup: ComputerSetup | undefined;
}

// What its owner lets the computer do and how much: three switches that save at once, runs at once, models, chat folder, labels.
export const ComputerSettingsCard = ({ computer, setup }: ComputerSettingsCardProps) => {
  const update = useUpdateComputerSettings(computer.id);

  return (
    <div className="@container divide-y divide-border rounded-lg border border-border px-4">
      <ComputerSwitchRow
        label="Plays you start"
        description="Runs pressed from a ticket, doc or chat"
        checked={!computer.plays_off}
        disabled={update.isPending}
        onChange={(on) => update.mutate({ plays_off: !on })}
      />
      <ComputerSwitchRow
        label="Auto plays"
        description="Runs that start by themselves"
        checked={!computer.auto_plays_off}
        disabled={update.isPending}
        onChange={(on) => update.mutate({ auto_plays_off: !on })}
      />
      <ComputerSwitchRow
        label="Terminal"
        description="Run commands and open a shell here from Nexul"
        checked={computer.shell_enabled}
        disabled={update.isPending}
        onChange={(on) => update.mutate({ shell_enabled: on })}
      />
      <ComputerRunsAtOnceRow computer={computer} />
      {setup && <ComputerModelsRow computer={computer} setup={setup} />}
      <ComputerChatFolderRow computer={computer} />
      <SettingsRow label="Labels" description="What a play's needs match" stacked className="last:pb-4">
        <ComputerLabels computer={computer} />
      </SettingsRow>
    </div>
  );
};
