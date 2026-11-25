import { KeyRound, RotateCcw } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { CopyButton } from "@/components/settings/CopyButton";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useRetryAuditStream, useRotateAuditSecret } from "@/ee/hooks/AuditLogHooks";
import type { AuditSettings } from "@/ee/models/AuditLog";
import { formatRelativeTime } from "@/utils/TimeUtility";

const lastSent = (s: AuditSettings) => (s.last_delivered_at ? `last sent ${formatRelativeTime(s.last_delivered_at)}` : "nothing sent yet");

// Where the queue stands: failed rows first, since they wait for an Owner, then rows still waiting, then up to date.
const DeliveriesStatus = ({ settings }: { settings: AuditSettings }) => (
  <span className="flex min-w-0 flex-col gap-1">
    {settings.dead > 0 && (
      <SettingsStatus tone="destructive" detail={settings.last_error}>
        {`${settings.dead} failed`}
      </SettingsStatus>
    )}
    {settings.pending > 0 && (
      <SettingsStatus tone="warning" detail={settings.dead === 0 ? (settings.last_error ?? lastSent(settings)) : lastSent(settings)}>
        {`${settings.pending} waiting`}
      </SettingsStatus>
    )}
    {settings.dead === 0 && settings.pending === 0 && (
      <SettingsStatus tone="success" detail={lastSent(settings)}>
        Up to date
      </SettingsStatus>
    )}
  </span>
);

const RetryButton = ({ unlocked }: { unlocked: boolean }) => {
  const retry = useRetryAuditStream();
  return (
    <ActionButton permission="instance:write" verb="retry failed deliveries" variant="outline" size="sm" icon={RotateCcw} loading={retry.isPending} disabled={!unlocked} onClick={() => retry.mutate()}>
      Retry failed
    </ActionButton>
  );
};

export const AuditDeliveriesRow = ({ settings }: { settings: AuditSettings }) => (
  <SettingsRow
    label="Deliveries"
    description="Each change is sent at least once and retried for a few minutes; one that still fails waits here until you retry it."
    stacked
  >
    <div className="flex flex-wrap items-center justify-between gap-3">
      <DeliveriesStatus settings={settings} />
      {settings.dead > 0 && <RetryButton unlocked={settings.unlocked} />}
    </div>
  </SettingsRow>
);

// A secret the server just made, shown this once; copy it into the receiver before leaving the page.
const SecretOnce = ({ secret }: { secret: string }) => (
  <span className="flex w-full min-w-0 items-center gap-1 rounded-md bg-surface-2 py-0.5 pr-0.5 pl-2.5 ring-1 ring-border">
    <code className="min-w-0 flex-1 truncate font-mono text-xs">{secret}</code>
    <CopyButton value={secret} label="Copy signing secret" iconOnly variant="ghost" />
  </span>
);

export const AuditNewSecretRow = ({ secret }: { secret: string }) => (
  <SettingsRow label="Your signing secret" description="Copy it into your receiver now. It is not shown again; Rotate secret makes a new one." stacked>
    <SecretOnce secret={secret} />
  </SettingsRow>
);

export const RotateSecretButton = ({ unlocked, onRotated }: { unlocked: boolean; onRotated: (secret: string) => void }) => {
  const rotate = useRotateAuditSecret();
  return (
    <ActionButton
      permission="instance:write"
      verb="rotate the signing secret"
      variant="ghost"
      size="sm"
      icon={KeyRound}
      loading={rotate.isPending}
      disabled={!unlocked}
      onClick={() => rotate.mutate(undefined, { onSuccess: (data) => data.new_secret && onRotated(data.new_secret) })}
    >
      Rotate secret
    </ActionButton>
  );
};
