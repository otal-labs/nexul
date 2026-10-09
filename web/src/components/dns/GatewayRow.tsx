import { CloudIcon, RouteIcon, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useDeleteGateway, useFetchExposures } from "@/hooks/DnsHooks";
import type { Gateway } from "@/models/DNS";

interface GatewayRowProps {
  gateway: Gateway;
}

const kindLabel = { tunnel: "Cloudflare tunnel", proxy: "Reverse proxy" } as const;

// What the gateway routes is its state: the hostnames on it, or none yet. Delete is refused by the backend while
// hostnames remain; that rejection surfaces through the mutation's toast.
export const GatewayRow = ({ gateway }: GatewayRowProps) => {
  const deleteGateway = useDeleteGateway();
  const { data: exposures } = useFetchExposures();
  const hostnames = exposures?.filter((exposure) => exposure.gateway_id === gateway.id).map((exposure) => exposure.hostname);
  const Icon = gateway.kind === "tunnel" ? CloudIcon : RouteIcon;

  return (
    <li className="flex items-start gap-3 px-3 py-3.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60">
        <Icon className="size-4 text-muted-foreground" aria-hidden />
      </span>
      <div className="min-w-0 flex-1 space-y-1">
        <p className="flex min-w-0 flex-wrap items-baseline gap-x-2">
          <span className="font-mono text-sm break-all">{gateway.docker_network}</span>
          <span className="text-xs text-muted-foreground">{kindLabel[gateway.kind]}</span>
        </p>
        {hostnames && hostnames.length > 0 && (
          <>
            <SettingsStatus tone="success">{hostnames.length === 1 ? "Routes 1 hostname" : `Routes ${hostnames.length} hostnames`}</SettingsStatus>
            <p className="font-mono text-xs break-all text-foreground/80">{hostnames.join("  ")}</p>
          </>
        )}
        {hostnames && hostnames.length === 0 && <SettingsStatus tone="muted">No hostnames yet</SettingsStatus>}
        <p className="flex min-w-0 flex-wrap gap-x-2 font-mono text-xs text-muted-foreground">
          {gateway.service_name && <span className="truncate">{gateway.service_name}</span>}
          {gateway.machine && <span className="truncate">on {gateway.machine}</span>}
          <span className="truncate">zone {gateway.zone}</span>
        </p>
      </div>
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel={`Delete gateway on ${gateway.docker_network}`}
        confirmLabel="Delete"
        loading={deleteGateway.isPending}
        onConfirm={() => deleteGateway.mutate(gateway.id)}
      />
    </li>
  );
};
