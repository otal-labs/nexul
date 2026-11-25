import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";

import { errorMessage } from "@/api/client";
import { FormCombobox } from "@/components/FormCombobox";
import { FormInput } from "@/components/FormInput";
import { TickerRow } from "@/components/TickerRow";
import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { SamlMetadataFields } from "@/ee/components/SamlMetadataFields";
import { useSaveSaml, useVerifySamlCheck } from "@/ee/hooks/SsoHooks";
import { MetadataSource, samlChecks, samlFormSchema, samlSaved, type SamlFormData, type SamlSettings } from "@/ee/models/Saml";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { useTicker } from "@/hooks/useTicker";
import { INSTANCE_WORKSPACE_ID } from "@/models/Workspace";

interface SamlProviderDialogProps {
  // The settings as saved, editing a provider when one is; their SP values say what to register before setup.
  saved: SamlSettings;
}

const formFrom = (saved: SamlSettings): SamlFormData => ({
  source: saved.metadata_xml_set ? MetadataSource.Xml : MetadataSource.Url,
  metadata_url: saved.metadata_url,
  metadata_xml: "",
  email_attribute: saved.email_attribute || "email",
  name_attribute: saved.name_attribute,
  display_name: saved.display_name,
  join_role_id: saved.join_role_id,
});

// As the OpenID Connect dialog: Verify ticks each check as its request lands; Confirm saves once every row is green.
export const SamlProviderDialog = ({ saved }: SamlProviderDialogProps) => {
  const editing = samlSaved(saved);
  const [open, setOpen] = useState(false);
  const save = useSaveSaml();
  const verifyCheck = useVerifySamlCheck();
  const roles = useFetchWorkspaceRoles(INSTANCE_WORKSPACE_ID);
  const defaults = formFrom(saved);
  const form = useForm<SamlFormData>({ defaultValues: defaults, resolver: zodResolver(samlFormSchema(saved.metadata_xml_set)) });
  const { fresh, verified, verifying, verify, reset, outcomeFor } = useTicker(samlChecks, useWatch({ control: form.control }), verifyCheck);
  const roleOptions = (roles.data ?? []).filter((r) => !r.is_owner_role).map((r) => ({ value: r.id, label: r.name }));

  const onConfirm = async (data: SamlFormData) => {
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
        <ActionButton permission="instance:write" verb="change single sign-on" size="sm" variant="outline">
          {editing ? "Edit provider" : "Set up SAML"}
        </ActionButton>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{editing ? "Edit the provider" : "Set up SAML single sign-on"}</DialogTitle>
          <DialogDescription>
            Register Nexul at your provider first, as entity ID {saved.sp.entity_id} with the assertion consumer service URL {saved.sp.acs_url}.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <form onSubmit={form.handleSubmit(verified ? onConfirm : verify)} className="space-y-4">
            <SamlMetadataFields keepsXml={saved.metadata_xml_set} />
            <FormInput control={form.control} name="email_attribute" id="saml_email_attribute" label="Email attribute" autoComplete="off" className="text-base" />
            <p className="text-xs text-muted-foreground">
              Signed by your provider, this address is trusted as theirs: if people can edit it there, they can sign in as someone else here.
            </p>
            <FormInput control={form.control} name="name_attribute" id="saml_name_attribute" label="Name attribute (optional)" autoComplete="off" className="text-base" />
            <FormInput control={form.control} name="display_name" id="saml_display_name" label="Name on the button" placeholder="Acme SSO" autoComplete="off" className="text-base" />
            <FormCombobox control={form.control} name="join_role_id" label="New people join as" options={roleOptions} placeholder="Choose a role" />
            <ul className="space-y-2" aria-label="Checks">
              {samlChecks.map((c) => (
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
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
};
