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
    <div className="mx-3 mt-3 flex items-center justify-between gap-3 border-t border-border pt-3 text-xs text-muted-foreground">
      <p className="min-w-0">
        {count === 1 ? "1 unfinished setup" : `${count} unfinished setups`}
        {newest && ` · started ${formatRelativeTime(newest)}`}
      </p>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="h-7 shrink-0 px-2 text-xs text-destructive hover:bg-destructive/10 hover:text-destructive"
        title="Removes the half-added computers; nothing ran on them"
        loading={clear.isPending}
        onClick={() => void clearAll()}
      >
        {count === 1 ? "Remove setup" : `Remove ${count} setups`}
      </Button>
    </div>
  );
};
