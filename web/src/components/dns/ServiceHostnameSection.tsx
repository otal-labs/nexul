import { CloudIcon, RouteIcon, Trash2 } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ExposeServiceDialog } from "@/components/dns/ExposeServiceDialog";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { NoFillBadge } from "@/components/ui/badge";
import { useDeleteExposure, useFetchExposures, useFetchGateways } from "@/hooks/DnsHooks";
import { useFetchConnectorStatus } from "@/hooks/ConnectorsHooks";
import type { Exposure, Gateway } from "@/models/DNS";
import type { Container } from "@/models/Stack";

interface ServiceHostnameSectionProps {
  containers: Container[];
}

interface ExposureRowProps {
  exposure: Exposure;
  gateway: Gateway | undefined;
  container: Container | undefined;
  removing: boolean;
  onRemove: () => void;
}

const ExposureRow = ({ exposure, gateway, container, removing, onRemove }: ExposureRowProps) => (
  <li className="flex flex-wrap items-center gap-x-3 gap-y-2 px-3 py-2.5">
    <a
      href={`https://${exposure.hostname}`}
      target="_blank"
      rel="noreferrer"
      className="break-all font-mono text-sm underline underline-offset-2"
    >
      {exposure.hostname}
    </a>
    {gateway && (
      <NoFillBadge icon={gateway.kind === "tunnel" ? CloudIcon : RouteIcon} color="text-muted-foreground">
        {gateway.kind}
      </NoFillBadge>
    )}
    <span className="font-mono text-xs text-muted-foreground">{container?.name ?? exposure.service}:{exposure.port}</span>
    <span className="ml-auto">
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel={`Unexpose ${exposure.hostname}`}
        confirmLabel="Unexpose"
        disabled={removing}
        onConfirm={onRemove}
      />
    </span>
  </li>
);

// Exposures route a hostname to one of the stack's containers, through a gateway resolved or provisioned by the
// backend (spec §7) — this section no longer picks a gateway itself, only the container and port.
export const ServiceHostnameSection = ({ containers }: ServiceHostnameSectionProps) => {
  const { data: cloudflare, isPending: statusPending } = useFetchConnectorStatus("cloudflare");
  const { data: gateways, isPending: gatewaysPending, error: gatewaysError } = useFetchGateways();
  const { data: exposures, isPending: exposuresPending, error: exposuresError } = useFetchExposures();
  const removeExposure = useDeleteExposure();

  const configured = cloudflare?.status.configured ?? false;
  const loading = statusPending || gatewaysPending || exposuresPending;
  const error = gatewaysError ?? exposuresError;
  const containerIds = new Set(containers.map((c) => c.id));
  const stackExposures = exposures?.filter((e) => !!e.service_id && containerIds.has(e.service_id)) ?? [];
  const gatewayById = new Map((gateways ?? []).map((g) => [g.id, g]));
  const containerById = new Map(containers.map((c) => [c.id, c]));
  const ready = configured && !loading;
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
      {error && <ErrorDisplay error={error} title="Could not load DNS state" />}
      {!statusPending && !configured && <EmptyRow>Connect a DNS provider to expose this stack.</EmptyRow>}
      {ready && containers.length === 0 && <EmptyRow>No containers parsed for this stack yet — nothing to expose.</EmptyRow>}
      {canExpose && stackExposures.length === 0 && <EmptyRow>Not exposed yet.</EmptyRow>}
      {ready && stackExposures.length > 0 && (
        <ul className="divide-y divide-border rounded-lg border border-border">
          {stackExposures.map((exposure) => (
            <ExposureRow
              key={exposure.id}
              exposure={exposure}
              gateway={gatewayById.get(exposure.gateway_id)}
              container={exposure.service_id ? containerById.get(exposure.service_id) : undefined}
              removing={removeExposure.isPending}
              onRemove={() => removeExposure.mutate(exposure.id)}
            />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
