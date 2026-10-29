import { ActionButton } from "@/components/action/ActionButton";
import { HistoryFooter } from "@/components/history/HistoryFooter";
import { useMergeAutomationVersion, useRollbackAutomationVersion } from "@/hooks/AutomationVersionHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { HistoryVersionSummary } from "@/models/History";

const NOTES = {
  active: "This is the active code",
  pending: "Merging makes it the active code; nothing is deleted",
  inactive: "Rolling back makes it the active code; nothing is deleted",
};

interface AutomationHistoryFooterProps {
  automationId: string;
  version: HistoryVersionSummary;
  onDone: () => void;
}

// An automation's versions are never restored: a pending one is merged, an older one is rolled back to, and the active one has nothing to do.
export const AutomationHistoryFooter = ({ automationId, version, onDone }: AutomationHistoryFooterProps) => {
  const merge = useMergeAutomationVersion(automationId);
  const rollback = useRollbackAutomationVersion(automationId);
  const { open: confirm } = useConfirmationDialog();
  const { id, status } = version;

  const onRollback = async () => {
    const ok = await confirm({
      title: "Roll back this version?",
      message: `#${version.version} becomes the active code and the automation's worker restarts.`,
      destructive: true,
    });
    if (ok && id) rollback.mutate(id, { onSuccess: onDone });
  };

  return (
    <HistoryFooter note={NOTES[status ?? "inactive"]}>
      {status === "pending" && id && (
        <ActionButton permission="automations:write" verb="merge automation versions" className="tabular-nums" loading={merge.isPending} onClick={() => merge.mutate(id, { onSuccess: onDone })}>
          Merge #{version.version}
        </ActionButton>
      )}
      {status === "inactive" && (
        <ActionButton permission="automations:write" verb="roll back automations" className="tabular-nums" loading={rollback.isPending} onClick={onRollback}>
          Rollback to #{version.version}
        </ActionButton>
      )}
    </HistoryFooter>
  );
};
