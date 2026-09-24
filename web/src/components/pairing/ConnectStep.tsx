import { ConnectionPanel } from "@/components/pairing/ConnectionPanel";
import { NameComputerForm } from "@/components/pairing/NameComputerForm";
import { TunnelChecks } from "@/components/pairing/TunnelChecks";
import { TunnelInstallSteps } from "@/components/pairing/TunnelInstallSteps";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchTunnelStatus } from "@/hooks/PairingHooks";
import { tunnelConnected, type Computer } from "@/models/Pairing";

interface TunnelConnectionProps {
  computer: Computer;
}

// The live signal beside the commands: the panel and its two checks, both driven by pushed status frames.
const TunnelConnection = ({ computer }: TunnelConnectionProps) => {
  const { data: status, error, isPending } = useFetchTunnelStatus(computer.id);
  return (
    <div className="space-y-3 @2xl:order-2">
      {isPending && <LoadingDisplay label="Reading the tunnel" />}
      {error && <ErrorDisplay error={error} title="Couldn't read the tunnel" />}
      {status && <ConnectionPanel connected={tunnelConnected(status)} hostname={computer.tunnel?.hostname ?? ""} />}
      {status && <TunnelChecks status={status} />}
    </div>
  );
};

interface ConnectStepProps {
  computer: Computer | undefined;
  onCreated: (computer: Computer) => void;
}

// Step one of pairing: name the computer, install its tunnel, and wait until the tunnel and the harness both answer.
export const ConnectStep = ({ computer, onCreated }: ConnectStepProps) => (
  <div className="@container">
    {!computer && <NameComputerForm onCreated={onCreated} />}
    {computer && (
      <div className="grid grid-cols-1 gap-5 @2xl:grid-cols-[minmax(0,1fr)_16rem]">
        <TunnelConnection computer={computer} />
        <TunnelInstallSteps computerId={computer.id} />
      </div>
    )}
  </div>
);
