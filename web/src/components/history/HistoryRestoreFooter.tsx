import { ActionButton } from "@/components/action/ActionButton";
import { HistoryFooter } from "@/components/history/HistoryFooter";
import type { HistoryRestore } from "@/models/History";
import { formatVersionTime } from "@/utils/TimeUtility";

interface HistoryRestoreFooterProps {
  version: number;
  savedAt: string;
  isCurrent: boolean;
  restore: HistoryRestore;
}

export const HistoryRestoreFooter = ({ version, savedAt, isCurrent, restore }: HistoryRestoreFooterProps) => (
  <HistoryFooter note={isCurrent ? "This is the current version" : "Restoring adds a new version; nothing is lost"}>
    <ActionButton
      permission={restore.permission}
      projectId={restore.projectId}
      answer={restore.answer}
      verb="restore versions"
      variant={isCurrent ? "outline" : "default"}
      disabled={isCurrent}
      loading={restore.pending}
      className="tabular-nums"
      onClick={() => restore.run(version)}
    >
      {isCurrent ? "Restore" : `Restore ${formatVersionTime(savedAt)}`}
    </ActionButton>
  </HistoryFooter>
);
