import { useState } from "react";

import { errorMessage } from "@/api/client";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProxyDeployWatch } from "@/components/setup/ProxyDeployWatch";
import { ProxyDomainForm } from "@/components/setup/ProxyDomainForm";
import { ProxyResolveWatch } from "@/components/setup/ProxyResolveWatch";
import { Button } from "@/components/ui/button";
import { useDeployInstanceProxy } from "@/hooks/DnsHooks";
import { useFetchPublicAddress } from "@/hooks/SetupHooks";
import type { ProxyDomainFormData } from "@/models/Setup";

// A new certificate is issued on the first HTTPS hit, so the finish keeps asking for about two minutes.
const CERTIFICATE_ATTEMPTS = 24;

interface SetupProxyPathProps {
  onFinish: (url: string, attempts: number) => void;
}

// Domain, wait for DNS, deploy Traefik, then the shared finish. A failed deploy remounts the DNS watch, which re-fires it.
export const SetupProxyPath = ({ onFinish }: SetupProxyPathProps) => {
  const { data: address, isPending, error } = useFetchPublicAddress();
  const deployProxy = useDeployInstanceProxy();
  const [target, setTarget] = useState<ProxyDomainFormData | null>(null);
  const [serviceId, setServiceId] = useState<string | null>(null);

  const deploy = () => {
    if (!target) return;
    deployProxy.mutate(target, { onSuccess: (result) => setServiceId(result.service_id) });
  };

  return (
    <div className="space-y-5">
      {isPending && <LoadingDisplay label="Looking up this server's public address" />}
      {error && <ErrorDisplay error={error} title="Couldn't find this server's public address." />}
      {address && !target && <ProxyDomainForm address={address} onSubmit={setTarget} />}
      {address && target && !serviceId && (
        <ProxyResolveWatch domain={target.domain} address={address} onResolved={deploy} />
      )}
      {deployProxy.isPending && <LoadingDisplay label="Starting the Traefik deploy" />}
      {deployProxy.isError && (
        <div className="space-y-3">
          <p role="alert" className="text-sm text-destructive">
            {errorMessage(deployProxy.error)}
          </p>
          <Button variant="outline" onClick={deploy}>
            Try again
          </Button>
        </div>
      )}
      {target && serviceId && (
        <ProxyDeployWatch
          serviceId={serviceId}
          onHealthy={() => onFinish(`https://${target.domain}`, CERTIFICATE_ATTEMPTS)}
          onRetry={() => setServiceId(null)}
        />
      )}
    </div>
  );
};
