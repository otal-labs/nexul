import { Plus } from "lucide-react";
import { useState } from "react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionButton } from "@/components/action/ActionButton";
import { AddRunnerForm } from "@/components/runner/AddRunnerForm";
import { RunnerInstallCommands } from "@/components/runner/RunnerInstallCommands";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useCreateRunnerEnrollment } from "@/hooks/RunnerHooks";
import { cn } from "@/lib/utils";
import type { Machine } from "@/models/Runner";

interface AddRunnerDialogProps {
  // Set from a machine's header: the runner joins that machine's pool, on the viewer's Manage of it; the label hides in a narrow @container.
  machine?: Pick<Machine, "name" | "actions">;
  triggerSize?: "default" | "sm";
  triggerVariant?: "default" | "outline" | "ghost" | "link";
  // Opened from a machine's ⋯, the dialog has no trigger of its own.
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

// Closing the dialog drops the enrollment, so reopening always starts from a fresh form and a fresh code.
export const AddRunnerDialog = ({ machine, triggerSize = "default", triggerVariant = "default", ...controlled }: AddRunnerDialogProps) => {
  const [ownOpen, setOwnOpen] = useState(false);
  const open = controlled.open ?? ownOpen;
  const machineName = machine?.name;
  const enroll = useCreateRunnerEnrollment();
  const onOpenChange = (next: boolean) => {
    setOwnOpen(next);
    controlled.onOpenChange?.(next);
    if (!next) enroll.reset();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {controlled.open === undefined && (
        <DialogTrigger asChild>
          <ActionButton
            permission={machine ? "machines:write" : "machines:create"}
            answer={machine && machineAnswer(machine, "machines:write")}
            verb="add runners"
            type="button"
            size={triggerSize}
            variant={triggerVariant}
            icon={Plus}
            className={cn(triggerVariant === "ghost" && "font-normal text-muted-foreground", triggerVariant === "link" && "h-auto items-baseline gap-1 p-0 text-row leading-5 has-[>svg]:px-0 [&_svg]:self-center")}
            {...(machineName && { "aria-label": "Add runner to this machine", title: "Add runner to this machine" })}
          >
            <span className={cn(machineName && "hidden @lg:inline")}>Add runner</span>
          </ActionButton>
        </DialogTrigger>
      )}
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add a runner</DialogTitle>
          <DialogDescription>
            {machineName
              ? `Name the runner, then run its install command on ${machineName}.`
              : "Name the runner, then run its install command on the machine that should build and deploy."}
          </DialogDescription>
        </DialogHeader>

        {!enroll.data && (
          <AddRunnerForm pending={enroll.isPending} onSubmit={(input) => enroll.mutateAsync({ ...input, machine: machineName })} />
        )}
        {enroll.data && <RunnerInstallCommands enrollment={enroll.data} name={enroll.variables?.name ?? ""} />}
      </DialogContent>
    </Dialog>
  );
};
