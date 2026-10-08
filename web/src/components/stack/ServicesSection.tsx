import { EnterList } from "@/components/EnterList";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ServiceRow } from "@/components/stack/ServiceRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchExposures } from "@/hooks/DnsHooks";
import { useFetchStackServices } from "@/hooks/StackHooks";
import type { Container } from "@/models/Stack";

interface ServicesSectionProps {
  stackId: string;
}

const summary = (services: Container[]): string => {
  const networks = [...new Set(services.flatMap((c) => (c.networks ?? []).map((n) => n.name)))];
  const observed = services.map((c) => c.observed_at ?? "").reduce((a, b) => (a > b ? a : b), "");
  const count = services.length === 1 ? "1 service" : `${services.length} services`;
  const where = networks.length > 0 ? ` on ${networks.join(", ")}` : "";
  const when = observed ? ` · observed ${formatRelativeTime(observed)}` : "";
  return count + where + when;
};

export const ServicesSection = ({ stackId }: ServicesSectionProps) => {
  const { data: services, isPending, error } = useFetchStackServices(stackId);
  const canSeeDns = useAreaAccess()?.("dns") ?? false;
  const { data: exposures } = useFetchExposures(canSeeDns);
  const hostnamesOf = (id: string) => (exposures ?? []).filter((e) => e.service_id === id).map((e) => e.hostname);

  return (
    <SettingsCard
      id="services"
      title="Services"
      description="Containers this stack declares, with what the runner last observed for each."
      footer={services && services.length > 0 && <p className="text-xs text-muted-foreground">{summary(services)}</p>}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Could not load services" />}
      {services && services.length === 0 && <EmptyRow flush>No services parsed for this stack yet.</EmptyRow>}
      {services && services.length > 0 && (
        <EnterList className="divide-y divide-border rounded-lg border border-border">
          {services.map((c) => (
            <ServiceRow key={c.id} container={c} hostnames={hostnamesOf(c.id)} />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
