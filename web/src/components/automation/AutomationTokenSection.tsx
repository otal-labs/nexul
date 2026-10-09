import { KeyRound } from "lucide-react";
import { useState } from "react";

import { AutomationTokenReveal } from "@/components/automation/AutomationTokenReveal";
import { Microheader } from "@/components/Microheader";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useMintAutomationToken, useRevokeAutomationToken } from "@/hooks/AutomationHooks";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Automation } from "@/models/Automation";
import { formatShortDate } from "@/utils/TimeUtility";

interface AutomationTokenSectionProps {
  automation: Automation;
}

// Rotating mints a fresh token (shown once, via the shared reveal) and retires the old one; revoking cuts access at once.
export const AutomationTokenSection = ({ automation }: AutomationTokenSectionProps) => {
  const mint = useMintAutomationToken(automation.id);
  const revoke = useRevokeAutomationToken(automation.id);
  const { data: catalog } = useFetchPermissionCatalog();
  const { open: confirm } = useConfirmationDialog();
  const [rotated, setRotated] = useState<string | null>(null);
  const revoked = !!automation.token_revoked_at;
  const labelOf = (scope: string) => catalog?.find((entry) => entry.value === scope)?.label;

  const onRotate = async () => {
    const ok = await confirm({
      title: "Rotate this token?",
      message: "The current token stops working now. The new one shows once, so copy it into the automation's code.",
      confirmLabel: "Rotate token",
      destructive: false,
    });
    if (ok) mint.mutate(undefined, { onSuccess: (result) => setRotated(result.token) });
  };

  const onRevoke = async () => {
    const ok = await confirm({
      title: "Revoke this token?",
      message: "The automation loses its access at once. Rotate later to give it a new token.",
      confirmLabel: "Revoke token",
    });
    if (ok) revoke.mutate();
  };

  return (
    <SettingsCard
      id="token"
      title="Token"
      description="How this automation's code signs in, and what it may do."
      aside={
        automation.token_prefix && (
          <SettingsStatus tone={revoked ? "destructive" : "success"}>
            {revoked && automation.token_revoked_at ? `Revoked ${formatShortDate(automation.token_revoked_at)}` : "Active"}
          </SettingsStatus>
        )
      }
      footer={
        <>
          <p className="text-xs text-muted-foreground">A new token shows once.</p>
          <div className="ml-auto flex items-center gap-1">
            {!revoked && (
              <Button type="button" variant="ghost" size="sm" className="hover:text-destructive" onClick={() => void onRevoke()} loading={revoke.isPending}>
                Revoke
              </Button>
            )}
            <Button type="button" variant="outline" size="sm" onClick={() => void onRotate()} loading={mint.isPending}>
              Rotate
            </Button>
          </div>
        </>
      }
    >
      <div className="space-y-5">
        {automation.token_prefix && (
          <div className="flex items-center gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-surface-2 text-muted-foreground">
              <KeyRound className="size-4" aria-hidden />
            </span>
            <div className="min-w-0">
              <p className="font-mono text-sm">dep_…{automation.token_prefix}</p>
            </div>
          </div>
        )}
        {rotated && <AutomationTokenReveal token={rotated} />}
        <section className="space-y-2">
          <Microheader>May do</Microheader>
          <ul className="divide-y divide-border rounded-md border border-border">
            {automation.scopes.map((scope) => (
              <li key={scope} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                {labelOf(scope) && <span className="min-w-0 truncate">{labelOf(scope)}</span>}
                <span className="shrink-0 font-mono text-xs text-muted-foreground">{scope}</span>
              </li>
            ))}
          </ul>
        </section>
      </div>
    </SettingsCard>
  );
};
