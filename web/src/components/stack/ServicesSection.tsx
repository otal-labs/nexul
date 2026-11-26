import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import { ServiceRow } from "@/components/stack/ServiceRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchExposures } from "@/hooks/DnsHooks";
import { useFetchStack, useFetchStackServices } from "@/hooks/StackHooks";
import { ContainerStatus, isServiceDown, type Container, type ServiceOutage } from "@/models/Stack";
import { deployModes, type DeployModes } from "@/utils/DeploySwapUtility";

interface ServicesSectionProps {
  stackId: string;
}

// Where the services run, beside the card's title; when the runner last looked is Health's to say.
const summary = (services: Container[]): string => {
  const networks = [...new Set(services.flatMap((c) => (c.networks ?? []).map((n) => n.name).filter(Boolean)))];
  const count = services.length === 1 ? "1 service" : `${services.length} services`;
  return networks.length > 0 ? `${count} on ${networks.join(", ")}` : count;
};

// How the next deploy treats one service, for its row's tooltip.
const deployNote = (modes: DeployModes, name: string): string => {
  if (modes.swapped.includes(name)) return "next deploy swaps it with no downtime";
  const restart = modes.restarted.find((g) => g.names.includes(name));
  return restart ? `next deploy restarts it in place (${restart.reason})` : "";
};

// What needs a look first: the services Health counts as down (the red ones before the stopped), then the ones on their
// way, then the healthy.
const rank = (c: Container, outages: ServiceOutage[] | undefined): number => {
  if (isServiceDown(c, outages)) return c.status === ContainerStatus.Stopped && !outages?.some((o) => o.service === c.name) ? 1 : 0;
  if (c.status === ContainerStatus.Healthy) return 3;
  return 2;
};

export const ServicesSection = ({ stackId }: ServicesSectionProps) => {
  const { data: services, isPending, error } = useFetchStackServices(stackId);
  const { data: stack } = useFetchStack(stackId);
  const canSeeDns = useAreaAccess()?.("dns") ?? false;
  const { data: exposures } = useFetchExposures(canSeeDns);
  const hostnamesOf = (id: string) => (exposures ?? []).filter((e) => e.service_id === id).map((e) => e.hostname);
  const outageOf = (c: Container) => stack?.outages?.find((o) => o.service === c.name);
  // Until the exposures load, or without dns:read, every service would read as having no hostname.
  const modes = services && exposures ? deployModes(services, hostnamesOf) : undefined;
  const sorted = [...(services ?? [])].sort((a, b) => rank(a, stack?.outages) - rank(b, stack?.outages));

  return (
    <SettingsCard
      id="services"
      title="Services"
      aside={services && services.length > 0 && <p className="font-mono text-xs text-muted-foreground">{summary(services)}</p>}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load services." />}
      {services && services.length === 0 && <EmptyRow flush>No services found in this stack yet.</EmptyRow>}
      {services && services.length > 0 && (
        <EnterList className={settingsListClass}>
          {sorted.map((c) => (
            <ServiceRow
              key={c.id}
              container={c}
              hostnames={hostnamesOf(c.id)}
              outage={outageOf(c)}
              deployNote={modes ? deployNote(modes, c.name) : ""}
            />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
