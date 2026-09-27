import { useState } from "react";

import { DnsStep } from "@/components/dns/DnsStep";
import { TunnelDeployStep } from "@/components/dns/TunnelDeployStep";
import { TunnelHostnameStep } from "@/components/dns/TunnelHostnameStep";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ManualConnectorDialog } from "@/components/settings/ManualConnectorDialog";
import { useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import { stepState, type TunnelDeployment } from "@/models/DNS";

interface SetupTunnelRungsProps {
  onFinish: (url: string) => void;
}

// The DNS page's tunnel rungs behind a Cloudflare token rung: before sign-in the token is the only way in.
export const SetupTunnelRungs = ({ onFinish }: SetupTunnelRungsProps) => {
  const { data: cloudflare, isPending, error } = useFetchConnectorStatus("cloudflare");
  const [tunnel, setTunnel] = useState<TunnelDeployment | null>(null);
  const [hostname, setHostname] = useState<string | null>(null);
  const connected = cloudflare?.status.configured ?? false;

  const onRouted = (host: string) => {
    setHostname(host);
    onFinish(`https://${host}`);
  };

  return (
    <>
      <DnsStep
        title="Connect Cloudflare"
        description="Paste an API token. Each permission is checked before it is saved."
        state={stepState(true, connected)}
        summary="Cloudflare connected"
      >
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {cloudflare && <ManualConnectorDialog connector={cloudflare.connector} />}
      </DnsStep>
      <DnsStep
        title="Deploy the tunnel"
        description="Nexul creates the tunnel at Cloudflare and runs cloudflared on this server until it connects."
        state={stepState(connected, tunnel !== null)}
        summary={tunnel && `Tunnel ${tunnel.tunnelName} connected · cloudflared on ${tunnel.target}`}
      >
        <TunnelDeployStep onConnected={setTunnel} />
      </DnsStep>
      <DnsStep
        title="Point your hostname"
        description="Routes the hostname into the tunnel and waits until it answers over HTTPS."
        state={stepState(tunnel !== null, hostname !== null)}
        summary={hostname}
      >
        {tunnel && <TunnelHostnameStep deployment={tunnel} onDone={(_, host) => onRouted(host)} />}
      </DnsStep>
    </>
  );
};
