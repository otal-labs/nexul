import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { DialogClose } from "@/components/ui/dialog";
import type { HistoryRestore } from "@/models/History";
import { formatVersionTime } from "@/utils/TimeUtility";

// The dialog's footer tray, opaque like the rest of the body.
export const footerTrayClass = "flex h-14 shrink-0 items-baseline gap-2 border-t border-border bg-[color-mix(in_oklab,var(--surface-2)_60%,var(--popover))] pt-2.5 pr-4 pl-6";

interface HistoryFooterProps {
  version: number;
  savedAt: string;
  isCurrent: boolean;
  restore: HistoryRestore;
}

export const HistoryFooter = ({ version, savedAt, isCurrent, restore }: HistoryFooterProps) => (
  <div className={footerTrayClass}>
    <p className="min-w-0 flex-1 truncate text-[13px] text-muted-foreground">
      {isCurrent ? "This is the current version" : "Restoring adds a new version; nothing is lost"}
    </p>
    <DialogClose asChild>
      <Button variant="ghost">Close</Button>
    </DialogClose>
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
  </div>
);
