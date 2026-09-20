import { CloudIcon, RouteIcon, Trash2 } from "lucide-react";

import { CreateGatewayDialog } from "@/components/dns/CreateGatewayDialog";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { NoFillBadge } from "@/components/ui/badge";
import { useDeleteGateway, useFetchGateways } from "@/hooks/DnsHooks";

// Lives under Settings → DNS; the onboarding stepper creates the first gateway without ever showing this list.
// Delete is refused by the backend while exposures still exist; that rejection surfaces through the mutation's toast.
export const GatewaysSection = () => {
  const { data: gateways, isPending, error } = useFetchGateways();
  const deleteGateway = useDeleteGateway();

  return (
    <SettingsCard
      id="gateways"
      title="Gateways"
      description="One gateway per docker network gives its services internet reachability."
    >
      <div className="mb-4 flex justify-end">
        <CreateGatewayDialog />
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Could not load gateways" />}
      {gateways && gateways.length === 0 && <NoDataDisplay message="No gateways yet." />}
      {gateways && gateways.length > 0 && (
        <ul className="space-y-2">
          {gateways.map((gateway) => (
            <li
              key={gateway.id}
              className="flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border bg-card p-3"
            >
              <NoFillBadge icon={gateway.kind === "tunnel" ? CloudIcon : RouteIcon} color="text-muted-foreground">
                {gateway.kind}
              </NoFillBadge>
              <span className="font-mono text-sm">{gateway.docker_network}</span>
              {gateway.service_name && (
                <span className="text-xs text-muted-foreground">{gateway.service_name}</span>
              )}
              <span className="text-xs text-muted-foreground">zone {gateway.zone}</span>
              <span className="ml-auto">
                <ConfirmDestroyButton
                  icon={Trash2}
                  idleLabel={`Delete gateway on ${gateway.docker_network}`}
                  confirmLabel="Delete"
                  disabled={deleteGateway.isPending}
                  onConfirm={() => deleteGateway.mutate(gateway.id)}
                />
              </span>
            </li>
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
