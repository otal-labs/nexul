import { useState } from "react";
import { Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { ActionButton } from "@/components/action/ActionButton";
import { useDeleteAutomationSecret } from "@/hooks/AutomationSecretHooks";
import { leavingRowClass } from "@/hooks/useRowGlide";
import { cn } from "@/lib/utils";
import type { AutomationSecretMeta } from "@/models/AutomationSecret";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface AutomationSecretRowProps {
  // Called as the row starts to leave, so the list can glide the rows under it up once it is gone.
  onLeave?: () => void;
  secret: AutomationSecretMeta;
  onReplace: () => void;
}

// Value is write-only, never shown again, matching GitHub Actions secrets (ADR 0047).
export const AutomationSecretRow = ({ onLeave, secret, onReplace }: AutomationSecretRowProps) => {
  const deleteSecret = useDeleteAutomationSecret();
  const [leaving, setLeaving] = useState(false);

  return (
    <li data-leaving={leaving || undefined} className={cn(leavingRowClass, "flex items-center justify-between gap-3 px-4 py-2.5")}>
      <div className="min-w-0">
        <p className="truncate font-mono text-sm font-medium">{secret.name}</p>
        <p className="font-mono text-xs text-muted-foreground">
          <span aria-hidden>••••••••</span> · updated {formatRelativeTime(secret.updated_at)}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <ActionButton permission="automations:write" verb="replace automation secrets" type="button" variant="ghost" size="sm" aria-label={`Replace ${secret.name}`} onClick={onReplace}>
          Replace
        </ActionButton>
        <ConfirmDestroyButton
          gate={{ permission: "automations:write", verb: "delete automation secrets" }}
          icon={Trash2}
          idleLabel={`Delete ${secret.name}`}
          confirmLabel="Delete"
          loading={deleteSecret.isPending}
          onConfirm={() => {
            setLeaving(true);
            onLeave?.();
            deleteSecret.mutate(secret.name, { onError: () => setLeaving(false) });
          }}
        />
      </div>
    </li>
  );
};
