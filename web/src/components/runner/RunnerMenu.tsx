import { MoreHorizontalIcon, Trash2 } from "lucide-react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { dropdownParts } from "@/components/listpane/rowMenu";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { useRemoveRunner } from "@/hooks/RunnerHooks";
import { useActionState } from "@/hooks/useActionState";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import type { Machine, Runner } from "@/models/Runner";

interface RunnerMenuProps {
  runner: Runner;
  // Its machine says what the viewer holds there; a runner with no machine yet answers to the instance.
  machine?: Pick<Machine, "actions"> | undefined;
}

// Holds Remove alone, so a viewer who isn't offered it gets an empty cell instead of a ⋯ that opens on nothing.
export const RunnerMenu = ({ runner, machine }: RunnerMenuProps) => {
  const gate = { permission: "machines:write", answer: machine && machineAnswer(machine, "machines:write"), verb: "remove runners" } as const;
  const offered = useActionState(gate).state !== "hidden";
  const remove = useRemoveRunner();
  const { open: confirm } = useConfirmationDialog();
  const name = runner.name || "this runner";

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

  if (!offered) return <span />;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="-my-1 size-7 self-center justify-self-end" aria-label={`More actions for ${name}`} loading={remove.isPending}>
          <MoreHorizontalIcon className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-40">
        <ActionMenuItem
          ui={dropdownParts}
          {...gate}
          icon={Trash2}
          destructive
          onSelect={() => void onRemove()}
        >
          Remove runner
        </ActionMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
