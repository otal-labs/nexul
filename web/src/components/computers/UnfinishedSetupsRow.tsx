import { CircleDashedIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useClearUnfinishedComputers, useListComputers } from "@/hooks/PairingHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { unfinishedComputers } from "@/models/SharedComputer";

// Add a computer saved a blank row before its install ran; the ones that never enrolled sit here until cleared.
export const UnfinishedSetupsRow = () => {
  const { data: computers } = useListComputers();
  const clear = useClearUnfinishedComputers();
  const { open: confirm } = useConfirmationDialog();
  const unfinished = unfinishedComputers(computers ?? []);
  if (unfinished.length === 0) return null;

  const count = unfinished.length;
  const clearAll = async () => {
    const ok = await confirm({
      title: count === 1 ? "Clear this setup?" : `Clear these ${count} setups?`,
      message: "Each was started and never finished installing, so nothing is paired. Add a computer starts a new one.",
      confirmLabel: "Clear",
    });
    if (ok) clear.mutate(unfinished);
  };

  return (
    <div className="flex items-center gap-2 px-3 py-2 text-xs text-muted-foreground">
      <CircleDashedIcon aria-hidden className="size-3.5 shrink-0" />
      <span className="min-w-0 flex-1">{count === 1 ? "1 setup never finished" : `${count} setups never finished`}</span>
      <Button type="button" variant="ghost" size="sm" className="h-6 px-2 text-xs underline" loading={clear.isPending} onClick={() => void clearAll()}>
        Clear
      </Button>
    </div>
  );
};
