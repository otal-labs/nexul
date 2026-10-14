import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { ActionButton } from "@/components/action/ActionButton";
import { useUpdateProviderOAuth } from "@/hooks/AuthHooks";
import { useActionState } from "@/hooks/useActionState";
import { oauthProviderCopy, oauthProviderFormSchema, type OAuthProviderFormData, type OptionalProvider } from "@/models/User";

interface OAuthProviderFormProps {
  provider: OptionalProvider;
  // The enabled provider's client ID when editing; its secret is never sent back, so blank keeps it.
  clientId?: string;
  onSaved?: () => void;
}

export const OAuthProviderForm = ({ provider, clientId, onSaved }: OAuthProviderFormProps) => {
  const copy = oauthProviderCopy[provider];
  const editing = clientId !== undefined;
  const update = useUpdateProviderOAuth(provider, copy.label);
  const writable = useActionState({ permission: "instance:write" }).state === "live";
  const form = useForm<OAuthProviderFormData>({
    disabled: !writable,
    defaultValues: { client_id: clientId ?? "", client_secret: "" },
    resolver: zodResolver(oauthProviderFormSchema(editing)),
  });

  const save = async (data: OAuthProviderFormData) => {
    try {
      await update.mutateAsync(data);
      form.reset({ client_id: data.client_id, client_secret: "" });
      onSaved?.();
    } catch {
      // The hook's toast shows the error; the form stays filled to retry.
    }
  };

  return (
    <form onSubmit={form.handleSubmit(save)} className="space-y-4">
      <FormInput
        control={form.control}
        name="client_id"
        id={`${provider}_client_id`}
        label="Client ID"
        placeholder={copy.idPlaceholder}
        autoComplete="off"
      />
      <FormInput
        control={form.control}
        name="client_secret"
        id={`${provider}_client_secret`}
        label="Client secret"
        type="password"
        placeholder={editing ? "Leave blank to keep the current secret" : copy.secretPlaceholder}
        autoComplete="new-password"
      />
      <ActionButton permission="instance:write" verb="change sign-in providers" type="submit" loading={form.formState.isSubmitting}>
        {editing ? "Save" : `Enable ${copy.label} sign-in`}
      </ActionButton>
    </form>
  );
};
