import { Plus } from "lucide-react";
import { useState } from "react";

import { AddRunnerForm } from "@/components/runner/AddRunnerForm";
import { RunnerInstallCommands } from "@/components/runner/RunnerInstallCommands";
import { Button } from "@/components/ui/button";
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

interface AddRunnerDialogProps {
  // Set from a machine's header: the runner joins that machine's pool; the label hides in a narrow @container.
  machineName?: string;
  triggerSize?: "default" | "sm";
  triggerVariant?: "default" | "outline" | "ghost";
}

// Closing the dialog drops the enrollment, so reopening always starts from a fresh form and a fresh code.
export const AddRunnerDialog = ({ machineName, triggerSize = "default", triggerVariant = "default" }: AddRunnerDialogProps) => {
  const [open, setOpen] = useState(false);
  const enroll = useCreateRunnerEnrollment();
  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (!next) enroll.reset();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button
          type="button"
          size={triggerSize}
          variant={triggerVariant}
          {...(machineName && { "aria-label": "Add a runner to this machine", title: "Add a runner to this machine" })}
        >
          <Plus className="size-4" aria-hidden />
          <span className={cn(machineName && "hidden @lg:inline")}>Add runner</span>
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add a runner</DialogTitle>
          <DialogDescription>
            {machineName
              ? `Name the runner, then run its install command on ${machineName}.`
              : "Name the runner, then run its install command on the machine that should build and deploy."}
          </DialogDescription>
        </DialogHeader>

        {!enroll.data && <AddRunnerForm machineName={machineName} pending={enroll.isPending} onSubmit={enroll.mutateAsync} />}
        {enroll.data && <RunnerInstallCommands enrollment={enroll.data} gitToken={enroll.variables?.gitToken ?? ""} />}
      </DialogContent>
    </Dialog>
  );
};
