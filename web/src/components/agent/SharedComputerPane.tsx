import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { ComputerCommands } from "@/components/settings/ComputerCommands";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { AGENT_PATH } from "@/models/Agent";
import { runnerPresence } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";
import { ComputerPermission, sharedComputerAbilities } from "@/models/SharedComputer";

interface SharedComputerPaneProps {
  computer: Computer;
}

// Someone else's computer shared with you: whether it is online and what its owner lets you do, nothing of their setup,
// and with Run commands the commands you ran there.
export const SharedComputerPane = ({ computer }: SharedComputerPaneProps) => {
  const wsPath = useWorkspacePath();
  const presence = runnerPresence(computer.runner);
  const shared = computer.shared;

  return (
    <Container className="space-y-6 py-8 sm:pr-3.5">
      <PageHeader
        crumbs={[{ label: "Agent", to: wsPath(AGENT_PATH) }, { label: "Shared with you" }]}
        title={shared ? `${shared.owner_name}'s ${computer.name}` : computer.name}
        meta={
          <SettingsStatus tone={presence.tone} detail={shared && sharedComputerAbilities(shared)}>
            {presence.text}
          </SettingsStatus>
        }
      />
      {shared?.permissions.includes(ComputerPermission.RunCommands) && (
        <dl className="rounded-md bg-surface-2 p-3">
          <ComputerCommands computer={computer} />
        </dl>
      )}
    </Container>
  );
};
