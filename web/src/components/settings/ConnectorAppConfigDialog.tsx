import { useState } from "react";

import { ConnectorAppConfigForm } from "@/components/settings/ConnectorAppConfigForm";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import type { Connector } from "@/models/Connectors";

interface ConnectorAppConfigDialogProps {
  connector: Connector;
}

// Replaces Connect while the connector's OAuth app is unregistered; saving refreshes the list so Connect takes over.
export const ConnectorAppConfigDialog = ({ connector }: ConnectorAppConfigDialogProps) => {
  const [open, setOpen] = useState(false);
  const callback = `${window.location.origin}/auth/connectors/${connector.id}/callback`;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">Set up app</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set up the {connector.name} app</DialogTitle>
          <DialogDescription>
            Register an OAuth app with {connector.name} using <code className="break-all">{callback}</code> as the
            redirect URL, then paste its credentials here. Nexul never holds or manages the app itself.
          </DialogDescription>
        </DialogHeader>
        <ConnectorAppConfigForm connectorId={connector.id} onSaved={() => setOpen(false)} />
      </DialogContent>
    </Dialog>
  );
};
