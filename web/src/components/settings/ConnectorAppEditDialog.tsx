import { useState } from "react";

import { ConnectorAppConfigForm } from "@/components/settings/ConnectorAppConfigForm";
import { ActionButton } from "@/components/action/ActionButton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import type { AppConfigStatus } from "@/models/Connectors";

interface ConnectorAppEditDialogProps {
  connectorId: string;
  current: AppConfigStatus;
}

// Changes a registered app in place; the card itself only shows what is registered.
export const ConnectorAppEditDialog = ({ connectorId, current }: ConnectorAppEditDialogProps) => {
  const [open, setOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <ActionButton permission="connectors:write" verb="edit connector apps" size="sm" variant="outline">
          Edit
        </ActionButton>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit the GitHub App</DialogTitle>
          <DialogDescription>
            Nexul checks each change with GitHub before saving. Leave the secret blank to keep the current one.
          </DialogDescription>
        </DialogHeader>
        <ConnectorAppConfigForm connectorId={connectorId} current={current} onSaved={() => setOpen(false)} />
      </DialogContent>
    </Dialog>
  );
};
