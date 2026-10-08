import { CloudIcon, RouteIcon, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { NoFillBadge } from "@/components/ui/badge";
import { useDeleteGateway } from "@/hooks/DnsHooks";
import type { Gateway } from "@/models/DNS";

interface GatewayRowProps {
  gateway: Gateway;
}

// Delete is refused by the backend while exposures still exist; that rejection surfaces through the mutation's toast.
export const GatewayRow = ({ gateway }: GatewayRowProps) => {
  const deleteGateway = useDeleteGateway();

  return (
    <li className="flex items-center gap-3 px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <NoFillBadge icon={gateway.kind === "tunnel" ? CloudIcon : RouteIcon} color="text-muted-foreground" className="shrink-0">
        {gateway.kind}
      </NoFillBadge>
      <div className="min-w-0 flex-1">
        <p className="font-mono text-sm break-all">{gateway.docker_network}</p>
        <p className="flex min-w-0 flex-wrap gap-x-2 text-xs text-muted-foreground">
          {gateway.service_name && <span className="truncate">{gateway.service_name}</span>}
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
