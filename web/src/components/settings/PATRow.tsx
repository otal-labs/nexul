import { Ban, Monitor, Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { useRevokePAT } from "@/hooks/AuthHooks";
import type { PersonalAccessToken } from "@/models/User";

interface PATRowProps {
  token: PersonalAccessToken;
}

// Revoking is irreversible, so it's gated behind ConfirmDestroyButton.
export const PATRow = ({ token }: PATRowProps) => {
  const revoke = useRevokePAT();

  return (
    <li className="flex items-center justify-between gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="min-w-0">
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
          {token.revoked_at && (
            <span className="inline-flex shrink-0 items-center gap-1 text-xs font-normal text-muted-foreground">
              <Ban className="size-3" aria-hidden />
              revoked
            </span>
          )}
        </p>
        <p className="font-mono text-xs text-muted-foreground tabular-nums">
          dep_…{token.prefix} · created {new Date(token.created_at).toLocaleDateString()}
          {token.last_used_at
            ? ` · last used ${new Date(token.last_used_at).toLocaleDateString()}`
            : ""}
        </p>
      </div>
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel="Revoke"
        loading={revoke.isPending}
        disabled={!!token.revoked_at}
        onConfirm={() => revoke.mutate(token.id)}
      />
    </li>
  );
};
