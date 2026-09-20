import { CheckCircle2, Loader2 } from "lucide-react";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import { useFetchTunnelStatus } from "@/hooks/DnsHooks";
import { useFetchServiceDeploys } from "@/hooks/ServiceHooks";
import type { TunnelDeployment } from "@/models/DNS";
import { latestDeploy } from "@/models/Service";

interface TunnelDeployWatchProps {
  deployment: TunnelDeployment;
  onConnected: (deployment: TunnelDeployment) => void;
  onRetry: () => void;
}

// Watches the runner's deploy and Cloudflare's connector state; the rung completes only once cloudflared is connected.
export const TunnelDeployWatch = ({ deployment, onConnected, onRetry }: TunnelDeployWatchProps) => {
  const { data: deploys } = useFetchServiceDeploys(deployment.serviceId, 3000);
  const { data: tunnel } = useFetchTunnelStatus(deployment.tunnelId);
  const deploy = latestDeploy(deploys);
  const failed = deploy?.status === "failed";
  const connected = tunnel?.status === "healthy";

  // Side effect on an external event (Cloudflare reporting the connector), not derived state.
  useEffect(() => {
    if (connected) onConnected(deployment);
  }, [connected, deployment, onConnected]);

  return (
    <div className="space-y-4">
      <p role="status" className="flex items-center gap-2 text-sm">
        {!failed && !connected && <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />}
        {connected && <CheckCircle2 className="size-4 text-success" aria-hidden />}
        {!deploy && !connected && !failed && `Sending cloudflared to ${deployment.target}…`}
        {deploy && !connected && !failed && deploy.status !== "healthy" && `Deploying cloudflared on ${deployment.target}…`}
        {deploy && !connected && !failed && deploy.status === "healthy" && "cloudflared is running, waiting for it to reach Cloudflare…"}
        {connected && `Tunnel ${deployment.tunnelName} is connected.`}
        {failed && `The deploy on ${deployment.target} failed.`}
      </p>
      {failed && deploy?.log && (
        <pre className="max-h-40 overflow-auto rounded-md border border-border bg-muted/40 p-3 font-mono text-xs text-muted-foreground">
          {deploy.log}
        </pre>
      )}
      {failed && (
        <Button variant="outline" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  );
};
