import { Blocks, Cloud, Mic } from "lucide-react";
import type { ComponentType } from "react";

import { personLabel } from "@nexul/client-core/person";

import { GithubMark } from "@/components/ProviderMarks";
import { ConnectorAppConfigDialog } from "@/components/settings/ConnectorAppConfigDialog";
import { ManualConnectorDialog } from "@/components/settings/ManualConnectorDialog";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { ActionButton } from "@/components/action/ActionButton";
import { useDisconnectConnector, useStartConnectorOAuth } from "@/hooks/ConnectorsHooks";
import { usePerson } from "@/hooks/PeopleHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { formatRelativeTime } from "@/utils/TimeUtility";
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
  const isManual = !!connector.manual?.length;
  const unconfigured = available && !status.configured;
  // OAuth connect needs the app registration first (CN3a); storing it takes connectors:write.
  const needsApp = unconfigured && !isManual && !app_configured;

  const onConnect = () => {
    startOAuth.mutate(connector.id, {
      onSuccess: (data) => window.location.assign(data.url),
    });
  };

  const { open: confirm } = useConfirmationDialog();
  const connectedBy = usePerson(status.connected_by ?? "");
  const connectedDetail = [
    status.connected_by && `by ${personLabel(connectedBy)}`,
    status.connected_at && formatRelativeTime(status.connected_at),
  ]
    .filter(Boolean)
    .join(" ");

  const onDisconnect = async () => {
    const ok = await confirm({
      title: `Disconnect ${connector.name}?`,
      message: `Nexul deletes the stored ${connector.name} credential and stops calling it until someone connects it again.`,
      confirmLabel: "Disconnect",
    });
    if (ok) disconnect.mutate(connector.id);
  };

  return (
    <li className="flex items-center gap-3 py-3.5">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60">
        <Icon className="size-4" />
      </span>
      <div className="min-w-0 flex-1 space-y-1">
        <p className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-0.5">
          <span className="text-sm font-medium">{connector.name}</span>
          {status.configured && (
            <SettingsStatus tone="success" detail={connectedDetail || undefined}>
              Connected
            </SettingsStatus>
          )}
          {unconfigured && <SettingsStatus tone="muted">Not connected</SettingsStatus>}
          {!available && <SettingsStatus tone="muted">Coming soon</SettingsStatus>}
        </p>
        <SettingsHint className="line-clamp-2">{connector.description}</SettingsHint>
      </div>
      {unconfigured && isManual && <ManualConnectorDialog connector={connector} />}
      {needsApp && <ConnectorAppConfigDialog connector={connector} />}
      {unconfigured && !isManual && app_configured && (
        <ActionButton permission="connectors:write" verb="connect connectors" size="sm" onClick={onConnect} loading={startOAuth.isPending}>
          Connect
        </ActionButton>
      )}
      {available && status.configured && (
        <ActionButton
          permission="connectors:write"
          verb="disconnect connectors"
          size="sm"
          variant="ghost"
          className="text-muted-foreground"
          liveClassName="hover:text-destructive"
          onClick={() => void onDisconnect()}
          loading={disconnect.isPending}
        >
          Disconnect
        </ActionButton>
      )}
    </li>
  );
};
