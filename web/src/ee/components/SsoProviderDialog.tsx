import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";

import { errorMessage } from "@/api/client";
import { FormCombobox } from "@/components/FormCombobox";
import { FormInput } from "@/components/FormInput";
import { SecretReferenceFieldTag } from "@/components/SecretReferenceFieldTag";
import { TickerRow } from "@/components/TickerRow";
import { ActionButton } from "@/components/action/ActionButton";
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
import { useSaveSso, useVerifySsoCheck } from "@/ee/hooks/SsoHooks";
import { ssoChecks, ssoFormSchema, type SsoFormData, type SsoSettings } from "@/ee/models/Sso";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useTicker } from "@/hooks/useTicker";
import { INSTANCE_WORKSPACE_ID } from "@/models/Workspace";

interface SsoProviderDialogProps {
  // The saved provider when editing; its secret is never sent back, so a blank secret keeps it.
  saved?: SsoSettings;
}

const emptyForm: SsoFormData = { issuer_url: "", client_id: "", client_secret: "", display_name: "", join_role_id: "" };

// Two beats, as a connector's: Verify ticks each check as its request lands; Confirm saves once every row is green.
export const SsoProviderDialog = ({ saved }: SsoProviderDialogProps) => {
  const editing = saved?.configured ?? false;
  const [open, setOpen] = useState(false);
  const save = useSaveSso();
  const verifyCheck = useVerifySsoCheck();
  const roles = useFetchWorkspaceRoles(INSTANCE_WORKSPACE_ID);
  const defaults = saved ? { ...emptyForm, issuer_url: saved.issuer_url, client_id: saved.client_id, display_name: saved.display_name, join_role_id: saved.join_role_id } : emptyForm;
  const form = useForm<SsoFormData>({ defaultValues: defaults, resolver: zodResolver(ssoFormSchema(editing)) });
  const { fresh, verified, verifying, verify, reset, outcomeFor } = useTicker(ssoChecks, useWatch({ control: form.control }), verifyCheck);

  const roleOptions = (roles.data ?? []).filter((r) => !r.is_owner_role).map((r) => ({ value: r.id, label: r.name }));

  const onConfirm = async (data: SsoFormData) => {
    try {
      await save.mutateAsync(data);
      setOpen(false);
      reset();
    } catch (err) {
      form.setError("root", { message: errorMessage(err) });
    }
  };

  const close = (next: boolean) => {
    setOpen(next);
    if (!next) {
      form.reset(defaults);
      reset();
    }
  };

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>
        <ActionButton permission="instance:write" verb="change single sign-on" size="sm" variant={editing ? "outline" : "default"}>
          {editing ? "Edit provider" : "Set up provider"}
        </ActionButton>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editing ? "Edit the provider" : "Set up single sign-on"}</DialogTitle>
          <DialogDescription>Register this instance with your OpenID Connect provider first, using the redirect URI on the page.</DialogDescription>
        </DialogHeader>
        <form onSubmit={form.handleSubmit(verified ? onConfirm : verify)} className="space-y-4">
          <FormInput control={form.control} name="issuer_url" id="sso_issuer_url" label="Issuer URL" placeholder="https://login.example.com" autoComplete="off" className="text-base" />
          <FormInput control={form.control} name="client_id" id="sso_client_id" label="Client ID" autoComplete="off" className="text-base" />
          <FormInput
            control={form.control}
            name="client_secret"
            id="sso_client_secret"
            label="Client secret"
            type="password"
            placeholder={editing ? "Leave blank to keep the current secret" : "A secret, or op://vault/item/field"}
            trailing={<SecretReferenceFieldTag control={form.control} name="client_secret" />}
            autoComplete="new-password"
            data-1p-ignore="true"
            data-lpignore="true"
            className="text-base"
          />
          <FormInput control={form.control} name="display_name" id="sso_display_name" label="Name on the button" placeholder="Acme SSO" autoComplete="off" className="text-base" />
          <FormCombobox control={form.control} name="join_role_id" label="New people join as" options={roleOptions} placeholder="Choose a role" />
          <ul className="space-y-2" aria-label="Checks">
            {ssoChecks.map((c) => (
              <TickerRow key={c.key} label={c.label} why={c.why} outcome={outcomeFor(c.key)} />
            ))}
          </ul>
          {form.formState.errors.root && <p className="text-sm text-destructive">{form.formState.errors.root.message}</p>}
          <DialogFooter>
            <Button type="submit" loading={verifying || form.formState.isSubmitting}>
              {verified && "Confirm"}
              {!verified && (fresh ? "Verify again" : "Verify")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
