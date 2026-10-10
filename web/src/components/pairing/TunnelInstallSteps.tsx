import { CommandSnippet } from "@/components/pairing/CommandSnippet";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchTunnelToken } from "@/hooks/PairingHooks";
import { PAIRING_GUIDE, tunnelCommands } from "@/utils/TunnelInstallCommands";

interface TunnelInstallStepsProps {
  computerId: string;
}

// The command that runs this computer's tunnel as a service with its connector token, and installs T3 Code if it's missing.
export const TunnelInstallSteps = ({ computerId }: TunnelInstallStepsProps) => {
  const { data: token, error, isPending } = useFetchTunnelToken(computerId);
  return (
    <div className="min-w-0 space-y-3">
      <p className="text-sm text-muted-foreground">Run this on the computer you're pairing. The token is secret, so keep it to that computer.</p>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {token && <CommandSnippet commands={tunnelCommands(token)} label="Tunnel command" guide={PAIRING_GUIDE} />}
    </div>
  );
};
