import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useRemoveRunner } from "@/hooks/RunnerHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Runner } from "@/models/Runner";

interface RemoveRunnerButtonProps {
  runner: Runner;
}

export const RemoveRunnerButton = ({ runner }: RemoveRunnerButtonProps) => {
  const remove = useRemoveRunner();
  const { open: confirm } = useConfirmationDialog();
  const name = runner.name || runner.id;

  const onRemove = async () => {
    const effect = runner.connected
      ? "It uninstalls itself now, and a job it is running fails."
      : "It uninstalls itself the next time it comes online.";
    const ok = await confirm({
      title: `Remove ${name}?`,
      message: `${effect} Its credential stops working. Add it again to bring it back.`,
      confirmLabel: "Remove runner",
    });
    if (ok) remove.mutate(runner.id);
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      aria-label={`Remove ${name}`}
      title={`Remove ${name}`}
      className="shrink-0 hover:text-destructive"
      loading={remove.isPending}
      onClick={() => void onRemove()}
    >
      <Trash2 className="size-4" />
    </Button>
  );
};
