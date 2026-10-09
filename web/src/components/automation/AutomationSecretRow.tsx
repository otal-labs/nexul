import { Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { Button } from "@/components/ui/button";
import { useDeleteAutomationSecret } from "@/hooks/AutomationSecretHooks";
import type { AutomationSecretMeta } from "@/models/AutomationSecret";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationSecretRowProps {
  secret: AutomationSecretMeta;
  onReplace: () => void;
}

// Value is write-only, never shown again, matching GitHub Actions secrets (ADR 0047).
export const AutomationSecretRow = ({ secret, onReplace }: AutomationSecretRowProps) => {
  const deleteSecret = useDeleteAutomationSecret();

  return (
    <li className="flex items-center justify-between gap-3 px-4 py-2.5">
      <div className="min-w-0">
        <p className="truncate font-mono text-sm font-medium">{secret.name}</p>
        <p className="font-mono text-xs text-muted-foreground">
          <span aria-hidden>••••••••</span> · updated {formatRelativeTime(secret.updated_at)}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <Button type="button" variant="ghost" size="sm" aria-label={`Replace ${secret.name}`} onClick={onReplace}>
          Replace
        </Button>
        <ConfirmDestroyButton
          icon={Trash2}
          idleLabel={`Delete ${secret.name}`}
          confirmLabel="Delete"
          loading={deleteSecret.isPending}
          onConfirm={() => deleteSecret.mutate(secret.name)}
        />
      </div>
    </li>
  );
};
