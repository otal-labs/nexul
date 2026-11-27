import { CircleDashedIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useClearUnfinishedComputers, useListComputers } from "@/hooks/PairingHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { unfinishedComputers } from "@/models/SharedComputer";
import { formatRelativeTime } from "@/utils/TimeUtility";

// Add a computer saved a blank row before its install ran; the ones that never enrolled sit here until cleared.
export const UnfinishedSetupsRow = () => {
  const { data: computers } = useListComputers();
  const clear = useClearUnfinishedComputers();
  const { open: confirm } = useConfirmationDialog();
  const unfinished = unfinishedComputers(computers ?? []);
  if (unfinished.length === 0) return null;

  const count = unfinished.length;
  const newest = unfinished.map((c) => c.created_at).sort().at(-1);
  const clearAll = async () => {
    const ok = await confirm({
      title: count === 1 ? "Remove this setup?" : `Remove these ${count} setups?`,
      message: "Each was started and never finished installing, so nothing is paired. Add a computer starts a new one.",
      confirmLabel: "Remove",
    });
    if (ok) clear.mutate(unfinished);
  };

  return (
    <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
      <CircleDashedIcon aria-hidden className="size-3.5 shrink-0" />
      <span className="min-w-0 flex-1">
        {count === 1 ? "1 setup never finished" : `${count} setups never finished`}
        {newest && ` · started ${formatRelativeTime(newest)}`}
      </span>
      <Button type="button" variant="ghost" size="sm" className="h-6 px-2 text-xs underline" title="Removes the half-added computers; nothing ran on them" loading={clear.isPending} onClick={() => void clearAll()}>
        Remove
      </Button>
    </div>
  );
};
