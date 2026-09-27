import { Loader2 } from "lucide-react";
import { useEffect, useRef } from "react";

import { Button } from "@/components/ui/button";
import { useFetchDeployLog } from "@/hooks/DeployHooks";
import { useFetchServiceDeploys } from "@/hooks/ServiceHooks";
import { latestDeploy } from "@/models/Service";

interface ProxyDeployWatchProps {
  serviceId: string;
  onHealthy: () => void;
  onRetry: () => void;
}

const FAILURE_TAIL_LINES = 5;

// Watches the gateway Traefik deploy the way TunnelDeployWatch watches cloudflared.
export const ProxyDeployWatch = ({ serviceId, onHealthy, onRetry }: ProxyDeployWatchProps) => {
  const { data: deploys } = useFetchServiceDeploys(serviceId, 3000);
  const deploy = latestDeploy(deploys);
  const failed = deploy?.status === "failed";
  const healthy = deploy?.status === "healthy";
  const { data: logLines } = useFetchDeployLog(failed ? deploy.id : undefined);
  const logTail = (logLines ?? []).slice(-FAILURE_TAIL_LINES).map((line) => line.text).join("\n");
  const fired = useRef(false);

  // Side effect on an external event (the runner reporting the deploy healthy), fired once.
  useEffect(() => {
    if (!healthy || fired.current) return;
    fired.current = true;
    onHealthy();
  }, [healthy, onHealthy]);

  return (
    <div className="space-y-4">
      <p role="status" className="flex items-center gap-2 text-sm">
        {!failed && <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />}
        {!failed && !healthy && "Deploying Traefik on ports 80 and 443…"}
        {healthy && "Traefik is running."}
        {failed && "The Traefik deploy failed."}
      </p>
      {failed && logTail && (
        <pre className="max-h-40 overflow-auto rounded-md border border-border bg-muted/40 p-3 font-mono text-xs text-muted-foreground">
          {logTail}
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
