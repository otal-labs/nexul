import { KeyRound } from "lucide-react";
import { useState } from "react";

import { AutomationTokenReveal } from "@/components/automation/AutomationTokenReveal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { useMintAutomationToken, useRevokeAutomationToken } from "@/hooks/AutomationHooks";
import type { Automation } from "@/models/Automation";

interface AutomationTokenSectionProps {
  automation: Automation;
}

// Rotating mints a fresh token (shown once, via the shared reveal); revoking kills access immediately.
export const AutomationTokenSection = ({ automation }: AutomationTokenSectionProps) => {
  const mint = useMintAutomationToken(automation.id);
  const revoke = useRevokeAutomationToken(automation.id);
  const [rotated, setRotated] = useState<string | null>(null);

  const onRotate = () => mint.mutate(undefined, { onSuccess: (result) => setRotated(result.token) });

  return (
    <section className="space-y-4 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold">Token</h2>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0 space-y-2">
          <div className="flex flex-wrap gap-1.5">
            {automation.scopes.map((scope) => (
              <Badge key={scope} variant="outline" className="font-mono">
                {scope}
              </Badge>
            ))}
          </div>
          {automation.token_prefix && (
            <p className="font-mono text-xs text-muted-foreground">dep_…{automation.token_prefix}</p>
          )}
          {automation.token_revoked_at && <p className="text-xs text-destructive">Revoked</p>}
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button type="button" variant="outline" size="sm" onClick={onRotate} disabled={mint.isPending}>
            {mint.isPending ? "Rotating…" : "Rotate"}
          </Button>
          <ConfirmDestroyButton
            icon={KeyRound}
            idleLabel="Revoke"
            disabled={!!automation.token_revoked_at || revoke.isPending}
            onConfirm={() => revoke.mutate()}
          />
        </div>
      </div>
      {rotated && <AutomationTokenReveal token={rotated} />}
    </section>
  );
};
