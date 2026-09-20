import { Trash2 } from "lucide-react";

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
        <p className="truncate text-sm font-medium">
          {token.name}
          {token.revoked_at && (
            <span className="ml-2 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
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
        disabled={!!token.revoked_at || revoke.isPending}
        onConfirm={() => revoke.mutate(token.id)}
      />
    </li>
  );
};
