import { PaneHeader } from "@/components/computers/PaneHeader";
import { ComputerCommands } from "@/components/settings/ComputerCommands";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useSetInOrder } from "@/hooks/PairingHooks";
import { runnerPresence } from "@/models/ComputerChecks";
import type { Computer } from "@/models/Pairing";
import { ComputerPermission, runsAgents, sharedComputerAbilities } from "@/models/SharedComputer";

interface SharedComputerPaneProps {
  computer: Computer;
}

// Someone else's computer: whether it is online and what its owner allows, never their setup.
export const SharedComputerPane = ({ computer }: SharedComputerPaneProps) => {
  const presence = runnerPresence(computer.runner);
  const shared = computer.shared;
  const setInOrder = useSetInOrder(computer.id);
  const inOrder = !!shared?.in_order;

  return (
    <div className="space-y-6">
      <PaneHeader
        mono
        title={shared ? `${shared.owner_name}'s ${computer.name}` : computer.name}
        meta={
          <SettingsStatus tone={presence.tone} detail={shared && sharedComputerAbilities(shared)}>
            {presence.text}
          </SettingsStatus>
        }
        actions={
          runsAgents(computer) && (
            <Button type="button" variant="outline" size="sm" loading={setInOrder.isPending} onClick={() => setInOrder.mutate(!inOrder)}>
              {inOrder ? "Remove from your order" : "Add to your order"}
            </Button>
          )
        }
      />
      {shared?.permissions.includes(ComputerPermission.RunCommands) && (
        <dl className="@container">
          <ComputerCommands computer={computer} />
        </dl>
      )}
    </div>
  );
};
