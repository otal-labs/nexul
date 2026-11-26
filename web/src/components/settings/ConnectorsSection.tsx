import { useEffect, useRef } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConnectorGroup } from "@/components/settings/ConnectorGroup";
import { SecretManagersStatus } from "@/components/settings/SecretManagersStatus";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchConnectors } from "@/hooks/ConnectorsHooks";

interface ConnectorsSectionProps {
  // Skips the SettingsCard wrapper here — the owner wizard already frames this step (T17).
  bare?: boolean;
}

// Self-contained (own fetch, no required props) so the owner wizard can drop it in directly.
export const ConnectorsSection = ({ bare = false }: ConnectorsSectionProps = {}) => {
  const { data, isPending, error } = useFetchConnectors();
  const [searchParams, setSearchParams] = useSearchParams();
  const toasted = useRef(false);

  // Toasts the OAuth callback once, then strips the query params so a refresh doesn't re-fire it.
  useEffect(() => {
    if (toasted.current || !data) return;
    const connectorId = searchParams.get("connector");
    const connected = searchParams.get("connected");
    const error = searchParams.get("error");
    if (!connectorId || (connected !== "1" && !error)) return;
    toasted.current = true;
    const name = data?.find((entry) => entry.connector.id === connectorId)?.connector.name ?? "Connector";
    if (error) toast.error(`${name}: ${error}`);
    if (!error) toast.success(`${name} connected`);
    const next = new URLSearchParams(searchParams);
    next.delete("connector");
    next.delete("connected");
    next.delete("error");
    setSearchParams(next, { replace: true });
  }, [data, searchParams, setSearchParams]);

  const connected = data?.filter((entry) => entry.status.configured) ?? [];
  const rest = data?.filter((entry) => !entry.status.configured) ?? [];

  const content = (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        // Connected first, the rest under their own label with Connect, so one list shows both without a filter.
        <div className="space-y-6">
          {connected.length > 0 && <ConnectorGroup entries={connected} />}
          {rest.length > 0 && <ConnectorGroup entries={rest} label={connected.length > 0 ? "Not connected" : undefined} />}
        </div>
      )}
    </>
  );

  if (bare) return content;

  return (
    <SettingsCard
      id="connectors"
      title="Connected tools"
      aside={<SecretManagersStatus />}
      description="Third-party tools this instance calls for you, each with one stored credential."
    >
      {content}
    </SettingsCard>
  );
};
