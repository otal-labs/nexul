import { useState } from "react";

import { OAuthProviderForm } from "@/components/settings/OAuthProviderForm";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { useUpdateProviderOAuth } from "@/hooks/AuthHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { oauthProviderCopy, type OptionalProvider } from "@/models/User";

interface OAuthProviderActionsProps {
  provider: OptionalProvider;
  clientId: string;
}

// An enabled provider's footer: Edit opens the form in a dialog; Disable asks first, as it stops sign-in this way.
export const OAuthProviderActions = ({ provider, clientId }: OAuthProviderActionsProps) => {
  const copy = oauthProviderCopy[provider];
  const [open, setOpen] = useState(false);
  const update = useUpdateProviderOAuth(provider, copy.label);
  const { open: confirm } = useConfirmationDialog();

  const disable = async () => {
    const ok = await confirm({
      title: `Disable ${copy.label} sign-in?`,
      message: `People who sign in with ${copy.label} can't until it is enabled again. Nobody is signed out.`,
      confirmLabel: "Disable",
      destructive: true,
    });
    if (ok) update.mutate({ client_id: "", client_secret: "" });
  };

  return (
    <>
      <Button variant="outline" size="sm" loading={update.isPending} onClick={() => void disable()}>
        Disable
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button size="sm">Edit</Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit {copy.label} sign-in</DialogTitle>
            <DialogDescription>Leave the secret blank to keep the current one.</DialogDescription>
          </DialogHeader>
          <OAuthProviderForm provider={provider} clientId={clientId} onSaved={() => setOpen(false)} />
        </DialogContent>
      </Dialog>
    </>
  );
};
