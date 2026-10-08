import { useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { toast } from "sonner";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { ConnectorCard } from "@/components/settings/ConnectorCard";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { cn } from "@/lib/utils";
import { useFetchConnectors } from "@/hooks/ConnectorsHooks";

type Tab = "connected" | "not-connected";

interface ConnectorsSectionProps {
  // Skips the SettingsCard wrapper here — the owner wizard already frames this step (T17).
  bare?: boolean;
}

// Self-contained (own fetch, no required props) so the owner wizard can drop it in directly.
export const ConnectorsSection = ({ bare = false }: ConnectorsSectionProps = {}) => {
  const { data, isPending, error } = useFetchConnectors();
  // Defaults to "Not connected" so Connect is visible before the owner wizard connects anything.
  const [pickedTab, setPickedTab] = useState<Tab | null>(null);
  const tab: Tab = pickedTab ?? (data?.some((entry) => entry.status.configured) ? "connected" : "not-connected");
  const [searchParams, setSearchParams] = useSearchParams();
  const toasted = useRef(false);

  // Toasts the OAuth callback once, then strips the query params so a refresh doesn't re-fire it.
  useEffect(() => {
    if (toasted.current) return;
    const connectorId = searchParams.get("connector");
    const connected = searchParams.get("connected");
    const error = searchParams.get("error");
    if (!connectorId || (connected !== "1" && !error)) return;
    toasted.current = true;
    const name = data?.find((entry) => entry.connector.id === connectorId)?.connector.name ?? connectorId;
    if (error) toast.error(`${name}: ${error}`);
    if (!error) toast.success(`${name} connected`);
    const next = new URLSearchParams(searchParams);
    next.delete("connector");
    next.delete("connected");
    next.delete("error");
    setSearchParams(next, { replace: true });
  }, [data, searchParams, setSearchParams]);

  const filtered = useMemo(
    () => data?.filter((entry) => (tab === "connected" ? entry.status.configured : !entry.status.configured)) ?? [],
    [data, tab],
  );

  const content = (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <div className="space-y-4">
          <div role="tablist" aria-label="Connectors" className="flex w-fit gap-1 rounded-md border p-1">
            {(["connected", "not-connected"] as const).map((t) => (
              <button
                key={t}
                type="button"
                role="tab"
                aria-selected={tab === t}
                onClick={() => setPickedTab(t)}
                className={cn(
                  "rounded px-3 py-1 text-sm font-medium transition-colors duration-150 ease-standard",
                  tab === t ? "bg-accent text-primary" : "text-muted-foreground hover:text-foreground",
                )}
              >
                {t === "connected" ? "Connected" : "Not connected"}
              </button>
            ))}
          </div>

          {filtered.length === 0 && <NoDataDisplay message="No connectors here" />}
          {filtered.length > 0 && (
            <ul className="divide-y divide-border overflow-hidden rounded-md border">
              {filtered.map((entry) => (
                <ConnectorCard key={entry.connector.id} entry={entry} />
              ))}
            </ul>
          )}
        </div>
      )}
    </>
  );

  if (bare) return content;

  return (
    <SettingsCard
      id="connectors"
      title="Connectors"
      description="Let this instance call out to third-party tools on your behalf."
    >
      {content}
    </SettingsCard>
  );
};
