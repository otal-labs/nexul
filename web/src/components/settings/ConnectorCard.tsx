import { Blocks, CheckCircle2, Cloud, Mic } from "lucide-react";
import type { ComponentType } from "react";

import { GithubMark } from "@/components/ProviderMarks";
import { ConnectorAppConfigDialog } from "@/components/settings/ConnectorAppConfigDialog";
import { ManualConnectorDialog } from "@/components/settings/ManualConnectorDialog";
import { NoFillBadge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useDisconnectConnector, useStartConnectorOAuth } from "@/hooks/ConnectorsHooks";
import type { ConnectorStatus } from "@/models/Connectors";

// Sole place mapping registry icon ids to lucide icons; falls back to Blocks when unmapped.
const iconById: Record<string, ComponentType<{ className?: string }>> = {
  github: GithubMark,
  cloudflare: Cloud,
  livekit: Mic,
};

interface ConnectorCardProps {
  entry: ConnectorStatus;
}

export const ConnectorCard = ({ entry }: ConnectorCardProps) => {
  const { connector, status, available, app_configured } = entry;
  const Icon = iconById[connector.icon] ?? Blocks;
  const startOAuth = useStartConnectorOAuth();
  const disconnect = useDisconnectConnector();
  const { data: me } = useFetchMe();
  const isOwner = me?.user?.can_create_workspace ?? false;
  const isManual = !!connector.manual?.length;
  const unconfigured = available && !status.configured;
  // OAuth connect needs the app registration first (CN3a); only an owner may store it.
  const needsApp = unconfigured && !isManual && !app_configured;

  const onConnect = () => {
    startOAuth.mutate(connector.id, {
      onSuccess: (data) => window.location.assign(data.url),
    });
  };

  const onDisconnect = () => disconnect.mutate(connector.id);

  return (
    <li className="flex items-center gap-3 p-4">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60">
        <Icon className="size-4" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="truncate text-sm font-medium">{connector.name}</p>
          {status.configured && (
            <NoFillBadge icon={CheckCircle2} color="text-success">
              Connected
            </NoFillBadge>
          )}
        </div>
        <p className="truncate text-sm text-muted-foreground">{connector.description}</p>
      </div>
      {!available && (
        <Button size="sm" variant="outline" disabled>
          Coming soon
        </Button>
      )}
      {unconfigured && isManual && <ManualConnectorDialog connector={connector} />}
      {needsApp && isOwner && <ConnectorAppConfigDialog connector={connector} />}
      {needsApp && !isOwner && (
        <Button size="sm" disabled title={`An owner has to set up the ${connector.name} app first`}>
          Connect
        </Button>
      )}
      {unconfigured && !isManual && app_configured && (
        <Button size="sm" onClick={onConnect} disabled={startOAuth.isPending}>
          Connect
        </Button>
      )}
      {available && status.configured && (
        <Button size="sm" variant="outline" onClick={onDisconnect} disabled={disconnect.isPending}>
          Disconnect
        </Button>
      )}
    </li>
  );
};
