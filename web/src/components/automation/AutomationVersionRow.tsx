import { RotateCcw } from "lucide-react";

import { AutomationVersionStatusBadge } from "@/components/automation/AutomationVersionStatusBadge";
import { Button } from "@/components/ui/button";
import { AutomationVersionStatus } from "@/enums/Automation";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { useRollbackAutomationVersion } from "@/hooks/AutomationVersionHooks";
import type { AutomationVersion } from "@/models/AutomationVersion";
import { formatRelativeTime } from "@/utils/TimeUtility";

// Mirrors internal/automations/versions.go's seedPusherID marker — never a
// real user id, so it's safe to special-case as a display label.
const SEED_PUSHER_ID = "nexul-seed";

interface AutomationVersionRowProps {
  automationId: string;
  version: AutomationVersion;
  canUpdate: boolean;
}

export const AutomationVersionRow = ({ automationId, version, canUpdate }: AutomationVersionRowProps) => {
  const rollback = useRollbackAutomationVersion(automationId);
  const { open: confirm } = useConfirmationDialog();
  const pusher = version.pusher_id === SEED_PUSHER_ID ? "Nexul" : version.pusher_id;
  const canRollback = canUpdate && version.status !== AutomationVersionStatus.Active;

  const onRollback = async () => {
    const ok = await confirm({
      title: "Roll back this version?",
      message: `This repoints the active code to #${version.sequence} and respawns the automation's worker.`,
      destructive: true,
    });
    if (ok) rollback.mutate(version.id);
  };

  return (
    <li className="flex flex-wrap items-center gap-3 px-4 py-3">
      <span className="font-mono text-sm">#{version.sequence}</span>
      <AutomationVersionStatusBadge status={version.status} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm">{version.message || "No message"}</p>
        <p className="truncate text-xs text-muted-foreground">
          {pusher} · {formatRelativeTime(version.created_at)}
        </p>
      </div>
      {canRollback && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onRollback}
          disabled={rollback.isPending}
        >
          <RotateCcw className="size-4" />
          Rollback
        </Button>
      )}
    </li>
  );
};
