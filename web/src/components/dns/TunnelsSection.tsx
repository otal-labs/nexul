import { EnterList } from "@/components/EnterList";
import { TunnelRow } from "@/components/dns/TunnelRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchTunnels } from "@/hooks/DnsHooks";

// Lives under Settings → DNS beside Gateways: the Cloudflare tunnels Nexul manages, each deletable once nothing uses it.
export const TunnelsSection = () => {
  const { data: tunnels, isPending, error } = useFetchTunnels();

  return (
    <SettingsCard
      id="tunnels"
      title="Tunnels"
      description="The Cloudflare tunnels Nexul manages. A tunnel can be deleted once nothing uses it: no gateway, hostname, instance address or cloudflared stack."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} title="Couldn't load tunnels." />}
      {tunnels && tunnels.length === 0 && <EmptyRow>No tunnels yet. Set up DNS creates one.</EmptyRow>}
      {tunnels && tunnels.length > 0 && (
        <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
          {tunnels.map((tunnel) => (
            <TunnelRow key={tunnel.id} tunnel={tunnel} />
          ))}
        </EnterList>
      )}
    </SettingsCard>
  );
};
