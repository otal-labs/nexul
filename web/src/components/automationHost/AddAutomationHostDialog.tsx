import { Plus } from "lucide-react";
import { useState } from "react";

import { AddAutomationHostForm } from "@/components/automationHost/AddAutomationHostForm";
import { AutomationHostInstallCommands } from "@/components/automationHost/AutomationHostInstallCommands";
import { ActionButton } from "@/components/action/ActionButton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useCreateAutomationHostEnrollment } from "@/hooks/AutomationHostHooks";

// Closing the dialog drops the enrollment, so reopening always starts from a fresh form and a fresh code.
export const AddAutomationHostDialog = () => {
  const [open, setOpen] = useState(false);
  const enroll = useCreateAutomationHostEnrollment();
  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (!next) enroll.reset();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <ActionButton permission="machines:write" verb="add automations hosts" icon={Plus} type="button" size="sm" variant="outline">
          Add automations host
        </ActionButton>
      </DialogTrigger>
      <DialogContent className="max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Add an automations host</DialogTitle>
          <DialogDescription>
            Name the host, then run its install command on the machine.
          </DialogDescription>
        </DialogHeader>

        {!enroll.data && <AddAutomationHostForm pending={enroll.isPending} onSubmit={enroll.mutateAsync} />}
        {enroll.data && <AutomationHostInstallCommands enrollment={enroll.data} />}
      </DialogContent>
    </Dialog>
  );
};
