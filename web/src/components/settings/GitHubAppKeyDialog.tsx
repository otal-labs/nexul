import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";

import { api, errorMessage } from "@/api/client";
import { TickerRow } from "@/components/TickerRow";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { useSetPrivateKey } from "@/hooks/ConnectorsHooks";
import { useTicker } from "@/hooks/useTicker";
import { privateKeyFormSchema, type CredentialCheck, type PrivateKeyFormData } from "@/models/Connectors";

const KEY_CHECKS: CredentialCheck[] = [
  {
    key: "key",
    label: "GitHub accepts the key as this App",
    why: "Nexul signs as the App with it to read every account the App is installed on, whoever connected GitHub.",
  },
];

interface GitHubAppKeyDialogProps {
  replacing: boolean;
}

// The ticker (Frontend Commandments): Verify asks GitHub, then Save stores the key, which never comes back.
export const GitHubAppKeyDialog = ({ replacing }: GitHubAppKeyDialogProps) => {
  const [open, setOpen] = useState(false);
  const setPrivateKey = useSetPrivateKey();
  const form = useForm<PrivateKeyFormData>({ defaultValues: { private_key: "" }, resolver: zodResolver(privateKeyFormSchema) });
  const { verified, verifying, verify, reset, outcomeFor } = useTicker(KEY_CHECKS, useWatch({ control: form.control }), (_, data) =>
    api.post("/api/connectors/github/private-key/verify", data),
  );

  const close = (next: boolean) => {
    setOpen(next);
    if (next) return;
    form.reset();
    reset();
  };

  const onSave = async ({ private_key }: PrivateKeyFormData) => {
    try {
      await setPrivateKey.mutateAsync({ id: "github", private_key });
      close(false);
    } catch (err) {
      form.setError("root", { message: errorMessage(err) });
    }
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>
        <Button size="sm" variant="outline">
          {replacing ? "Replace private key" : "Add private key"}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{replacing ? "Replace the private key" : "Add the App's private key"}</DialogTitle>
          <DialogDescription>
            On GitHub, open your App's settings, then General → Private keys → Generate a private key, and paste the
            whole .pem file here.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(verified ? onSave : verify)} className="space-y-4">
          <Controller
            control={form.control}
            name="private_key"
            render={({ field, fieldState }) => (
              <div className="space-y-1.5">
                <Textarea
                  {...field}
                  aria-label="Private key"
                  aria-invalid={!!fieldState.error}
                  placeholder="-----BEGIN RSA PRIVATE KEY-----"
                  autoComplete="off"
                  spellCheck={false}
                  data-1p-ignore="true"
                  data-lpignore="true"
                  className="h-40 font-mono text-xs"
                />
                {fieldState.error && <p className="text-sm text-destructive">{fieldState.error.message}</p>}
              </div>
            )}
          />
          <ul className="space-y-2" aria-label="Private key check">
            {KEY_CHECKS.map((c) => (
              <TickerRow key={c.key} label={c.label} why={c.why} outcome={outcomeFor(c.key)} />
            ))}
          </ul>
          {form.formState.errors.root && (
            <p role="alert" className="text-sm text-destructive">
              {form.formState.errors.root.message}
            </p>
          )}
          <DialogFooter>
            <Button type="submit" loading={form.formState.isSubmitting || verifying}>
              {form.formState.isSubmitting && verified && "Saving…"}
              {verifying && "Verifying…"}
              {!form.formState.isSubmitting && !verifying && (verified ? "Save" : "Verify")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
