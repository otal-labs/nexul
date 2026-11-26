import { Blocks, Cloud, Mic } from "lucide-react";
import type { ComponentType } from "react";

import { personLabel } from "@nexul/client-core/person";
import { secretManagerName } from "@nexul/client-core/secretReference";

import { GithubMark } from "@/components/ProviderMarks";
import { ConnectorAppConfigDialog } from "@/components/settings/ConnectorAppConfigDialog";
import { ManualConnectorDialog } from "@/components/settings/ManualConnectorDialog";
import { destroyActionClass, destroyActionLiveClass } from "@/components/settings/ConfirmDestroyButton";
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
  const connectorBy = status.connected_by && personLabel(connectedBy);
  const connectedDetail = [
    // The owner wizard connects as "setup", which is no person, so it says only when.
    connectorBy && connectorBy !== "setup" && `by ${connectorBy}`,
    status.connected_at && formatRelativeTime(status.connected_at),
  ]
    .filter(Boolean)
    .join(", ");

  const onDisconnect = async () => {
    const ok = await confirm({
      title: `Disconnect ${connector.name}?`,
      message: `Nexul deletes the stored ${connector.name} credential and stops calling it until someone connects it again.`,
      confirmLabel: "Disconnect",
    });
    if (ok) disconnect.mutate(connector.id);
  };

  return (
    <li className="flex min-h-14 items-center gap-3 py-2.5">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60">
        <Icon className="size-4" />
      </span>
      {/* The name with when it was connected, what it is for under it, and a status only for a problem. */}
      <div className="min-w-0 flex-1 space-y-0.5">
        <p className="flex min-w-0 flex-wrap items-baseline gap-x-2">
          <span className="text-sm font-medium">{connector.name}</span>
          {status.configured && !status.secret && connectedDetail && (
            <span className="text-xs text-muted-foreground">Connected {connectedDetail}</span>
          )}
        </p>
        <p className="text-xs text-pretty text-muted-foreground">{connector.description}</p>
        {!available && <SettingsStatus tone="muted">Coming soon</SettingsStatus>}
        {status.secret && (
          <SettingsStatus tone={status.secret.state === "unreachable" ? "warning" : "destructive"} detail={status.secret.error}>
            {status.secret.state ? `${secretManagerName(status.secret.manager)} ${status.secret.state}` : "Secret not resolved"}
          </SettingsStatus>
        )}
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
          className={destroyActionClass}
          liveClassName={destroyActionLiveClass}
          onClick={() => void onDisconnect()}
          loading={disconnect.isPending}
        >
          Disconnect
        </ActionButton>
      )}
    </li>
  );
};
