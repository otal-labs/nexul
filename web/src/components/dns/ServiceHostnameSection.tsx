import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ExposeServiceDialog } from "@/components/dns/ExposeServiceDialog";
import { ExposureRow } from "@/components/dns/ExposureRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchExposures, useFetchGateways } from "@/hooks/DnsHooks";
import { useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import type { Container } from "@/models/Stack";

interface ServiceHostnameSectionProps {
  containers: Container[];
}

// Exposures route a hostname to one of the stack's containers, through a gateway resolved or provisioned by the
// backend (spec §7) — this section no longer picks a gateway itself, only the container and port.
export const ServiceHostnameSection = ({ containers }: ServiceHostnameSectionProps) => {
  const { data: cloudflare, isPending: statusPending } = useFetchConnectorStatus("cloudflare");
  const { data: gateways, isPending: gatewaysPending, error: gatewaysError } = useFetchGateways();
  const { data: exposures, isPending: exposuresPending, error: exposuresError } = useFetchExposures();

  const configured = cloudflare?.status.configured ?? false;
  const loading = statusPending || gatewaysPending || exposuresPending;
  const error = gatewaysError ?? exposuresError;
  const containerIds = new Set(containers.map((c) => c.id));
  const stackExposures = exposures?.filter((e) => !!e.service_id && containerIds.has(e.service_id)) ?? [];
  const gatewayById = new Map((gateways ?? []).map((g) => [g.id, g]));
  const containerById = new Map(containers.map((c) => [c.id, c]));
  const ready = configured && !loading && !error;
  const canExpose = ready && containers.length > 0;

  return (
    <SettingsCard
      id="exposures"
      title="Exposures"
      description="Hostnames routed to this stack's containers through a tunnel or reverse proxy."
      footer={
        canExpose ? (
          <>
            <p className="text-xs text-muted-foreground">
              {stackExposures.length === 0 && "Pick a container and a hostname to route to it."}
              {stackExposures.length === 1 && "1 hostname routed to this stack."}
              {stackExposures.length > 1 && `${stackExposures.length} hostnames routed to this stack.`}
            </p>
            <ExposeServiceDialog containers={containers} />
          </>
        ) : undefined
      }
    >
      {loading && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load DNS." />}
      {!statusPending && !configured && <EmptyRow flush>Connect a DNS provider to expose this stack.</EmptyRow>}
      {ready && containers.length === 0 && <EmptyRow flush>No containers in this stack yet, so nothing to expose.</EmptyRow>}
      {canExpose && stackExposures.length === 0 && <EmptyRow flush>Not exposed yet.</EmptyRow>}
      {ready && stackExposures.length > 0 && (
        <EnterList className="divide-y divide-border rounded-md border border-border">
          {stackExposures.map((exposure) => (
            <ExposureRow
              key={exposure.id}
              exposure={exposure}
              gateway={gatewayById.get(exposure.gateway_id)}
              container={exposure.service_id ? containerById.get(exposure.service_id) : undefined}
            />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
