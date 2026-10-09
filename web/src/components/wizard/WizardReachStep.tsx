import { zodResolver } from "@hookform/resolvers/zod";
import { CheckCircle2, Loader2 } from "lucide-react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useShallow } from "zustand/react/shallow";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ExposureTicker } from "@/components/wizard/ExposureTicker";
import { FormSelect } from "@/components/ticket/FormSelect";
import { Button } from "@/components/ui/button";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
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
  port: z.coerce.number<number>().int().positive("Enter a port above 0"),
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
      {zonesError && <ErrorDisplay error={zonesError} title="Couldn't load zones." />}
      {servicesError && <ErrorDisplay error={servicesError} title="Couldn't load the stack's services." />}
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
  const [subdomain, zone] = useWatch({ control: form.control, name: ["subdomain", "zone"] });
  const hostname = fullHostname(subdomain, zone);

  const exposed = createExposure.data;
  const via = exposed && gatewayName(exposed.gateway_id);
  const exposedLabel = exposed && (via ? `${exposed.hostname} via a ${via} gateway` : exposed.hostname);

  const onSubmit = (data: ReachFormData) =>
    createExposure
      .mutateAsync({
        hostname: fullHostname(data.subdomain, data.zone),
        service_id: data.service_id,
        port: data.port,
        zone_id: data.zone_id,
        zone: data.zone,
      })
      .catch(() => undefined); // the ticker shows the server's message; exposure creation is retry-safe

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
      <ExposureTicker status={createExposure.status} error={createExposure.error} result={exposedLabel} />
      <WizardFooter skip={!exposed && <WizardSkipLink onClick={onSkip} />}>
        {exposed && (
          <Button type="button" onClick={() => onDone(exposed.id, exposedLabel ?? exposed.hostname)}>
            Continue
          </Button>
        )}
        {!exposed && (
          <Button type="submit" loading={createExposure.isPending}>
            Expose service
          </Button>
        )}
      </WizardFooter>
    </form>
  );
};
