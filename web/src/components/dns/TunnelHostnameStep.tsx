import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useFetchDnsZones, useRouteTunnelHostname } from "@/hooks/DnsHooks";
import { fullHostname, soleItem, type DnsSetupResult, type TunnelDeployment, type Zone } from "@/models/DNS";

const TunnelHostnameSchema = z.object({
  subdomain: z.string().trim(),
  zone_id: z.string().min(1, "Choose a zone"),
  zone: z.string().min(1, "Choose a zone"),
  service: z.string().trim().min(1, "Local service is required"),
});

type TunnelHostnameFormData = z.infer<typeof TunnelHostnameSchema>;

interface TunnelHostnameFieldsProps {
  deployment: TunnelDeployment;
  zones: Zone[];
  onDone: (result: DnsSetupResult) => void;
}

// Mounted only once zones are loaded so defaultValues can preselect the sole zone.
const TunnelHostnameFields = ({ deployment, zones, onDone }: TunnelHostnameFieldsProps) => {
  const routeTunnel = useRouteTunnelHostname();
  const soleZone = soleItem(zones);
  const form = useForm<TunnelHostnameFormData>({
    defaultValues: { subdomain: "", zone_id: soleZone?.id ?? "", zone: soleZone?.name ?? "", service: "http://web:80" },
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
      onDone({
        headline: `${host} routes through the ${deployment.tunnelName} tunnel to this instance.`,
        detail: `${host} → ${data.service} · cloudflared on ${deployment.target}`,
      });
    } catch {
      // Errors surface through the hook's toast; routing is retry-safe.
    }
  };

  return (
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
      <FormInput
        control={form.control}
        name="service"
        label="Local service (cloudflared forwards here)"
        placeholder="http://web:80"
      />
      {hostname && (
        <p className="font-mono text-xs text-muted-foreground">
          {hostname} → {service}
        </p>
      )}
      <Button type="submit" className="w-full sm:w-auto" disabled={routeTunnel.isPending}>
        {routeTunnel.isPending ? "Routing hostname…" : "Point hostname at the tunnel"}
      </Button>
    </form>
  );
};

interface TunnelHostnameStepProps {
  deployment: TunnelDeployment;
  onDone: (result: DnsSetupResult) => void;
}

export const TunnelHostnameStep = ({ deployment, onDone }: TunnelHostnameStepProps) => {
  const { data: zones, isPending, error } = useFetchDnsZones(true);

  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {zones && <TunnelHostnameFields deployment={deployment} zones={zones} onDone={onDone} />}
    </>
  );
};
