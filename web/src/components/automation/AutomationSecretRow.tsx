import { Trash2 } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { useDeleteAutomationSecret } from "@/hooks/AutomationSecretHooks";
import type { AutomationSecretMeta } from "@/models/AutomationSecret";

interface AutomationSecretRowProps {
  secret: AutomationSecretMeta;
}

// Value is write-only — never shown again, matching GitHub Actions secrets (ADR 0047).
export const AutomationSecretRow = ({ secret }: AutomationSecretRowProps) => {
  const deleteSecret = useDeleteAutomationSecret();

  return (
    <li className="flex items-center justify-between gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <div className="min-w-0">
        <p className="truncate font-mono text-sm font-medium">{secret.name}</p>
        <p className="font-mono text-xs text-muted-foreground">
          updated {new Date(secret.updated_at).toLocaleDateString()}
        </p>
      </div>
      <ConfirmDestroyButton
        icon={Trash2}
        idleLabel="Delete"
        disabled={deleteSecret.isPending}
        onConfirm={() => deleteSecret.mutate(secret.name)}
      />
    </li>
  );
};
