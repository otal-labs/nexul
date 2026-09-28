import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useUpdateProviderOAuth } from "@/hooks/AuthHooks";
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
  const form = useForm<OAuthProviderFormData>({
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
      <Button type="submit" disabled={form.formState.isSubmitting}>
        {form.formState.isSubmitting && "Saving…"}
        {!form.formState.isSubmitting && (editing ? "Save" : `Enable ${copy.label} sign-in`)}
      </Button>
    </form>
  );
};
