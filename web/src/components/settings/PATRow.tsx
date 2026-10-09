import { Monitor, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { useRevokePAT } from "@/hooks/AuthHooks";
import type { PersonalAccessToken } from "@/models/User";
import { formatRelativeTime, formatShortDate } from "@/utils/TimeUtility";

interface PATRowProps {
  token: PersonalAccessToken;
}

// Revoking is irreversible, so it's gated behind ConfirmDestroyButton; a revoked token stays listed, greyed by its status.
export const PATRow = ({ token }: PATRowProps) => {
  const revoke = useRevokePAT();
  const lastUsed = token.last_used_at ? `last used ${formatRelativeTime(token.last_used_at)}` : "never used";

  return (
    <li className="flex items-center justify-between gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="min-w-0 space-y-1">
        <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
          <span className="truncate" title={token.name}>
            {token.name}
          </span>
          {token.computer_id && (
            <span className="inline-flex shrink-0 items-center gap-1 text-xs font-normal text-muted-foreground">
              <Monitor className="size-3" aria-hidden />
              paired computer
            </span>
          )}
        </p>
        <p className="flex min-w-0">
          {token.revoked_at && <SettingsStatus tone="muted">Revoked {formatShortDate(token.revoked_at)}</SettingsStatus>}
          {!token.revoked_at && <SettingsStatus tone="success" detail={lastUsed}>Active</SettingsStatus>}
        </p>
        <p className="truncate font-mono text-xs text-muted-foreground tabular-nums">
          dep_…{token.prefix} · created {formatShortDate(token.created_at)}
        </p>
      </div>
      {!token.revoked_at && (
        <ConfirmDestroyButton icon={Trash2} idleLabel="Revoke" loading={revoke.isPending} onConfirm={() => revoke.mutate(token.id)} />
      )}
    </li>
  );
};
