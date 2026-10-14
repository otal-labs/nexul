import { Trash2 } from "lucide-react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { useRemoveRunner } from "@/hooks/RunnerHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Machine, Runner } from "@/models/Runner";

interface RemoveRunnerButtonProps {
  runner: Runner;
  // Its machine's card says what the viewer holds there; a runner with no machine yet answers to the instance.
  machine?: Pick<Machine, "actions"> | undefined;
}

export const RemoveRunnerButton = ({ runner, machine }: RemoveRunnerButtonProps) => {
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
    <ActionButton
      permission="machines:write"
      answer={machine && machineAnswer(machine, "machines:write")}
      verb="remove runners"
      type="button"
      variant="ghost"
      size="icon"
      icon={Trash2}
      aria-label={`Remove ${name}`}
      title={`Remove ${name}`}
      className="size-8 shrink-0"
      liveClassName="hover:text-destructive"
      loading={remove.isPending}
      onClick={() => void onRemove()}
    />
  );
};
