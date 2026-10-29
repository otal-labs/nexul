import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";
import type { HistoryRestore } from "@/models/History";
import { formatVersionTime } from "@/utils/TimeUtility";

interface HistoryFooterProps {
  version: number;
  /** When the version before it was saved; none on the oldest kept. */
  since: string | undefined;
  isCurrent: boolean;
  restore: HistoryRestore;
}

export const HistoryFooter = ({ version, since, isCurrent, restore }: HistoryFooterProps) => (
  <div className="flex items-center gap-2 border-t border-border bg-surface-2/60 py-3 pr-4 pl-6">
    <p className="min-w-0 flex-1 text-sm text-muted-foreground">
      {since ? `Changes since ${formatVersionTime(since)}` : "The oldest version kept"}
    </p>
    <DialogClose asChild>
      <Button variant="outline">Close</Button>
    </DialogClose>
    <ActionButton
      permission={restore.permission}
      projectId={restore.projectId}
      answer={restore.answer}
      verb="restore versions"
      disabled={isCurrent}
      loading={restore.pending}
      onClick={() => restore.run(version)}
    >
      Restore this version
    </ActionButton>
  </div>
);
