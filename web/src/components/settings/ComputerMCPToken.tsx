import { KeyRound, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { CommandBlock } from "@/components/pairing/CommandBlock";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { useFetchMCPToken, useMintMCPToken, useRevokeMCPToken } from "@/hooks/PairingHooks";

interface ComputerMCPTokenProps {
  computerId: string;
}

// The raw token lives only on the mint mutation's data, so a revoke or a reload drops it for good.
export const ComputerMCPToken = ({ computerId }: ComputerMCPTokenProps) => {
  const { data: token, isPending, error } = useFetchMCPToken(computerId);
  const mint = useMintMCPToken(computerId);
  const revoke = useRevokeMCPToken(computerId);
  const busy = mint.isPending || revoke.isPending;

  return (
    <div className="space-y-2">
      {isPending && <LoadingDisplay label="Loading MCP token" className="justify-start p-0" />}
      {error && <ErrorDisplay error={error} title="MCP token unavailable" className="p-3" />}
      {token === null && (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="text-xs text-muted-foreground">No MCP token — its providers can't reach Nexul's MCP server.</p>
          <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => mint.mutate()}>
            <KeyRound className="size-4" />
            Mint MCP token
          </Button>
        </div>
      )}
      {token && (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="min-w-0 truncate font-mono text-xs text-muted-foreground tabular-nums">
            {token.name} · dep_…{token.prefix} · created {new Date(token.created_at).toLocaleDateString()}
            {token.last_used_at && ` · last used ${new Date(token.last_used_at).toLocaleDateString()}`}
          </p>
          <span className="flex shrink-0 items-center gap-1">
            <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => mint.mutate()}>
              <KeyRound className="size-4" />
              Replace
            </Button>
            <ConfirmDestroyButton
              icon={Trash2}
              idleLabel="Revoke MCP token"
              confirmLabel="Revoke"
              disabled={busy}
              onConfirm={() => revoke.mutate(undefined, { onSuccess: () => mint.reset() })}
            />
          </span>
        </div>
      )}
      {mint.data && <CommandBlock shell="MCP token · shown once" lines={[mint.data.token]} prompt="" />}
    </div>
  );
};
