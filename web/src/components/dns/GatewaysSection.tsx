import { Link } from "react-router";

import { CreateGatewayDialog } from "@/components/dns/CreateGatewayDialog";
import { GatewayRow } from "@/components/dns/GatewayRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchGateways } from "@/hooks/DnsHooks";

// Lives under Settings → DNS; the onboarding stepper creates the first gateway without ever showing this list.
export const GatewaysSection = () => {
  const { data: gateways, isPending, error } = useFetchGateways();

  return (
    <SettingsCard
      id="gateways"
      title="Gateways"
      description="One gateway per docker network gives its services internet reachability. To put a hostname on a service, use the DNS wizard."
      footer={
        <>
          <Button asChild variant="outline" size="sm">
            <Link to="/wizard/onboarding/dns">Set up DNS</Link>
          </Button>
          <CreateGatewayDialog />
        </>
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Could not load gateways" />}
      {gateways && gateways.length === 0 && <EmptyRow>No gateways yet.</EmptyRow>}
      {gateways && gateways.length > 0 && (
        <ul className="divide-y divide-border overflow-hidden rounded-md border">
          {gateways.map((gateway) => (
            <GatewayRow key={gateway.id} gateway={gateway} />
          ))}
        </ul>
      )}
    </SettingsCard>
  );
};
