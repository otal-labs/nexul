import { RotateCcw } from "lucide-react";

import { AutomationVersionStatusBadge } from "@/components/automation/AutomationVersionStatusBadge";
import { ActionButton } from "@/components/action/ActionButton";
import { AutomationVersionStatus } from "@/enums/Automation";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { usePersonNamer } from "@/hooks/usePersonNamer";
import { useRollbackAutomationVersion } from "@/hooks/AutomationVersionHooks";
import type { AutomationVersion } from "@/models/AutomationVersion";
import { formatRelativeTime } from "@/utils/TimeUtility";

// Mirrors internal/automations/versions.go's seedPusherID marker — never a
// real user id, so it's safe to special-case as a display label.
const SEED_PUSHER_ID = "nexul-seed";

interface AutomationVersionRowProps {
  automationId: string;
  version: AutomationVersion;
}

export const AutomationVersionRow = ({ automationId, version }: AutomationVersionRowProps) => {
  const rollback = useRollbackAutomationVersion(automationId);
  const { open: confirm } = useConfirmationDialog();
  const nameOf = usePersonNamer();
  const pusher = version.pusher_id === SEED_PUSHER_ID ? "Nexul" : nameOf(version.pusher_id);
  const canRollback = version.status === AutomationVersionStatus.Inactive;

  const onRollback = async () => {
    const ok = await confirm({
      title: "Roll back this version?",
      message: `#${version.sequence} becomes the active code and the automation's worker restarts.`,
      destructive: true,
    });
    if (ok) rollback.mutate(version.id);
  };

  return (
    <li className="grid grid-cols-[2.5rem_4.5rem_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-2.5 transition-colors duration-150 ease-standard hover:bg-accent/40">
      <span className="font-mono text-sm tabular-nums">#{version.sequence}</span>
      <AutomationVersionStatusBadge status={version.status} />
      <div className="min-w-0">
        <p className="truncate text-sm">{version.message || "No message"}</p>
        <p className="truncate text-xs text-muted-foreground">
          {pusher} · {formatRelativeTime(version.created_at)}
        </p>
      </div>
      {canRollback && (
        <ActionButton
          permission="automations:write"
          verb="roll back automations"
          icon={RotateCcw}
          type="button"
          variant="outline"
          size="sm"
          onClick={onRollback}
          loading={rollback.isPending}
        >
          Rollback
        </ActionButton>
      )}
    </li>
  );
};
