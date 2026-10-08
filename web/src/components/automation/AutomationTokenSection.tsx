import { KeyRound } from "lucide-react";
import { useState } from "react";

import { AutomationTokenReveal } from "@/components/automation/AutomationTokenReveal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
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
    <SettingsCard
      id="token"
      title="Token"
      description="What this automation's token may do. Rotate shows a new token once. Revoke cuts its access at once."
      footer={
        <div className="flex items-center gap-1">
          <Button type="button" variant="outline" size="sm" onClick={onRotate} loading={mint.isPending}>
            Rotate
          </Button>
          <ConfirmDestroyButton
            icon={KeyRound}
            idleLabel="Revoke"
            loading={revoke.isPending}
            disabled={!!automation.token_revoked_at}
            onConfirm={() => revoke.mutate()}
          />
        </div>
      }
    >
      <div className="space-y-4">
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
        {rotated && <AutomationTokenReveal token={rotated} />}
      </div>
    </SettingsCard>
  );
};
