import { zodResolver } from "@hookform/resolvers/zod";
import { Loader2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { AdvancedFields } from "@/components/dns/AdvancedFields";
import { TunnelRouteChecks } from "@/components/dns/TunnelRouteChecks";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useFetchDnsZones, useRouteTunnelHostname, verifyTunnelCheck } from "@/hooks/DnsHooks";
import { useTicker } from "@/hooks/useTicker";
import {
  TUNNEL_ROUTE_CHECKS,
  fullHostname,
  soleItem,
  zonesInAccount,
  type DnsSetupResult,
  type TunnelDeployment,
  type Zone,
} from "@/models/DNS";
import { retry } from "@/utils/RetryUtility";

// A fresh CNAME can take a minute to answer, so the reachable row keeps asking before it goes red.
const REACHABLE_ATTEMPTS = 20;
const REACHABLE_RETRY_MS = 3000;

const TunnelHostnameSchema = z.object({
  subdomain: z.string().trim(),
  zone_id: z.string().min(1, "Choose a zone"),
  zone: z.string().min(1, "Choose a zone"),
  service: z.string().trim(),
});

type TunnelHostnameFormData = z.infer<typeof TunnelHostnameSchema>;

interface TunnelHostnameFieldsProps {
  deployment: TunnelDeployment;
  zones: Zone[];
  onDone: (result: DnsSetupResult, hostname: string) => void;
}

// Mounted only once zones are loaded so defaultValues can preselect the sole zone.
const TunnelHostnameFields = ({ deployment, zones, onDone }: TunnelHostnameFieldsProps) => {
  const routeTunnel = useRouteTunnelHostname();
  const [routed, setRouted] = useState<{ host: string; result: DnsSetupResult } | null>(null);
  const ticker = useTicker(TUNNEL_ROUTE_CHECKS, { hostname: routed?.host }, (key) =>
    key === "reachable"
      ? retry(REACHABLE_ATTEMPTS, REACHABLE_RETRY_MS, () => verifyTunnelCheck(deployment.tunnelId, key))
      : verifyTunnelCheck(deployment.tunnelId, key),
  );
  const soleZone = soleItem(zones);
  const form = useForm<TunnelHostnameFormData>({
    defaultValues: { subdomain: "", zone_id: soleZone?.id ?? "", zone: soleZone?.name ?? "", service: "" },
    resolver: zodResolver(TunnelHostnameSchema),
  });

  const zoneName = (id: string) => zones.find((z) => z.id === id)?.name ?? "";
  const hostname = fullHostname(form.watch("subdomain"), form.watch("zone"));
  const service = form.watch("service");

  const onSubmit = async (data: TunnelHostnameFormData) => {
    const host = fullHostname(data.subdomain, data.zone);
    try {
      await routeTunnel.mutateAsync({
        tunnel_id: deployment.tunnelId,
        hostname: host,
        zone_id: data.zone_id,
        zone: data.zone,
        service: data.service,
      });
      setRouted({
        host,
        result: {
          headline: `${host} routes through the ${deployment.tunnelName} tunnel to this instance.`,
          detail: `${host} → ${data.service || "this instance"} · cloudflared on ${deployment.target}`,
        },
      });
      await ticker.verify({ hostname: host });
    } catch {
      // Errors surface through the hook's toast; routing is retry-safe.
    }
  };

  return (
    <>
      {routed && (
        <TunnelRouteChecks
          hostname={routed.host}
          outcomeFor={ticker.outcomeFor}
          verified={ticker.verified}
          verifying={ticker.verifying}
          onCheckAgain={() => void ticker.verify({ hostname: routed.host })}
          onContinue={() => onDone(routed.result, routed.host)}
        />
      )}
      {!routed && (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <FormInput control={form.control} name="subdomain" label="Subdomain" placeholder="app" />
        <FormSelect
          control={form.control}
          name="zone_id"
          label="Zone"
          placeholder="Choose a zone…"
          options={zones.map((z) => ({ value: z.id, label: z.name }))}
          onChangeValue={(value) => form.setValue("zone", zoneName(value), { shouldValidate: true })}
        />
      </div>
      {hostname && (
        <p className="font-mono text-xs text-muted-foreground">
          {hostname} → {service || "this instance"}
        </p>
      )}
      <AdvancedFields>
        <FormInput
          control={form.control}
          name="service"
          label="Forward to a different local service"
          placeholder="http://my-app:3000"
        />
      </AdvancedFields>
      <Button type="submit" className="w-full sm:w-auto" disabled={routeTunnel.isPending}>
        {routeTunnel.isPending && <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />}
        {routeTunnel.isPending ? "Routing hostname…" : "Point hostname at the tunnel"}
      </Button>
    </form>
      )}
    </>
  );
};

interface TunnelHostnameStepProps {
  deployment: TunnelDeployment;
  onDone: (result: DnsSetupResult, hostname: string) => void;
}

export const TunnelHostnameStep = ({ deployment, onDone }: TunnelHostnameStepProps) => {
  const { data: zones, isPending, error } = useFetchDnsZones(true);

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {zones && (
        <TunnelHostnameFields
          deployment={deployment}
          zones={zonesInAccount(zones, deployment.accountId)}
          onDone={onDone}
        />
      )}
    </>
  );
};
