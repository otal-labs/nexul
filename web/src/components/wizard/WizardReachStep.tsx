import { zodResolver } from "@hookform/resolvers/zod";
import { CheckCircle2, Loader2 } from "lucide-react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useShallow } from "zustand/react/shallow";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { useCreateServiceExposure, useFetchDnsZones, useFetchGateways } from "@/hooks/DnsHooks";
import { useFetchStackDeploys, useFetchStackServices } from "@/hooks/StackHooks";
import { fullHostname } from "@/models/DNS";
import { latestDeploy } from "@/models/Stack";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

const ReachFormSchema = z.object({
  subdomain: z.string().trim(),
  zone_id: z.string().min(1, "Choose a zone"),
  zone: z.string().min(1, "Choose a zone"),
  service_id: z.string().min(1, "Choose a service"),
  port: z.coerce.number<number>().int().positive("Port must be positive"),
});
type ReachFormData = z.infer<typeof ReachFormSchema>;

// Ambient status while the owner fills in the form below — same "watch the deploy while you work" idea as
// TunnelDeployWatch, without gating the form on it (reach is optional and the deploy may already be healthy).
const DeployStatusLine = ({ stackId }: { stackId: string }) => {
  const { data: deploys } = useFetchStackDeploys(stackId, 3000);
  const deploy = latestDeploy(deploys);
  const healthy = deploy?.status === "healthy";
  return (
    <p role="status" className="flex items-center gap-2 text-xs text-muted-foreground">
      {!healthy && <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" aria-hidden />}
      {healthy && <CheckCircle2 className="size-3.5 text-success" aria-hidden />}
      {deploy ? `Deploy is ${deploy.status}.` : "Deploying…"}
    </p>
  );
};

interface WizardReachStepProps {
  onDone: () => void;
  onSkip: () => void;
}

// Zone + subdomain + service/port, defaulted to the candidate's reachable service (spec §7 issue answer); the
// server reuses or provisions the gateway, so the result just names which one it picked.
export const WizardReachStep = ({ onDone, onSkip }: WizardReachStepProps) => {
  const { stackId, candidate } = useProjectWizardStore(
    useShallow((s) => ({ stackId: s.stackId, candidate: s.candidate })),
  );
  const setExposure = useProjectWizardStore((s) => s.setExposure);
  const { data: zones, isPending: zonesPending, error: zonesError } = useFetchDnsZones();
  const { data: services, isPending: servicesPending, error: servicesError } = useFetchStackServices(
    stackId ?? undefined,
  );
  const { data: gateways } = useFetchGateways();
  const createExposure = useCreateServiceExposure();

  const defaultService = services?.find((c) => c.name === candidate?.reachable?.service) ?? services?.[0];
  const defaultPort = candidate?.reachable?.port ?? Number(defaultService?.declared.ports?.[0] ?? 0);

  return (
    <>
      {(zonesPending || servicesPending) && <LoadingDisplay />}
      {zonesError && <ErrorDisplay error={zonesError} title="Could not load zones" />}
      {servicesError && <ErrorDisplay error={servicesError} title="Could not load the stack's services" />}
      {zones && services && stackId && (
        <ReachForm
          zones={zones}
          services={services}
          defaultServiceId={defaultService?.id ?? ""}
          defaultPort={defaultPort}
          stackId={stackId}
          createExposure={createExposure}
          onDone={(exposureId, hostname) => {
            setExposure(exposureId, hostname);
            onDone();
          }}
          onSkip={onSkip}
          gatewayName={(gatewayId: string) => gateways?.find((g) => g.id === gatewayId)?.kind}
        />
      )}
    </>
  );
};

interface ReachFormProps {
  zones: { id: string; name: string }[];
  services: { id: string; name: string }[];
  defaultServiceId: string;
  defaultPort: number;
  stackId: string;
  createExposure: ReturnType<typeof useCreateServiceExposure>;
  onDone: (exposureId: string, hostname: string) => void;
  onSkip: () => void;
  gatewayName: (gatewayId: string) => string | undefined;
}

const ReachForm = ({
  zones,
  services,
  defaultServiceId,
  defaultPort,
  stackId,
  createExposure,
  onDone,
  onSkip,
  gatewayName,
}: ReachFormProps) => {
  const form = useForm<ReachFormData>({
    defaultValues: { subdomain: "", zone_id: "", zone: "", service_id: defaultServiceId, port: defaultPort },
    resolver: zodResolver(ReachFormSchema),
  });
  const zoneName = (id: string) => zones.find((z) => z.id === id)?.name ?? "";
  const hostname = fullHostname(form.watch("subdomain"), form.watch("zone"));

  const onSubmit = async (data: ReachFormData) => {
    const host = fullHostname(data.subdomain, data.zone);
    try {
      const exposure = await createExposure.mutateAsync({
        hostname: host,
        service_id: data.service_id,
        port: data.port,
        zone_id: data.zone_id,
        zone: data.zone,
      });
      const via = gatewayName(exposure.gateway_id);
      onDone(exposure.id, via ? `${host} via a ${via} gateway` : host);
    } catch {
      // Errors surface through the hook's toast; exposure creation is retry-safe.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
      <DeployStatusLine stackId={stackId} />
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
      <div className="grid gap-4 sm:grid-cols-2">
        <FormSelect
          control={form.control}
          name="service_id"
          label="Service"
          options={services.map((c) => ({ value: c.id, label: c.name }))}
        />
        <FormInput control={form.control} name="port" label="Port" type="number" />
      </div>
      {hostname && <p className="font-mono text-xs text-muted-foreground">{hostname}</p>}
      <div className="flex flex-wrap gap-3">
        <Button type="submit" disabled={createExposure.isPending}>
          {createExposure.isPending ? "Exposing…" : "Expose service"}
        </Button>
        <Button type="button" variant="ghost" className="text-muted-foreground" onClick={onSkip}>
          Skip for now
        </Button>
      </div>
    </form>
  );
};
