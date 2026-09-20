import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useUpdateProviderOAuth } from "@/hooks/AuthHooks";
import {
  OAuthProviderFormSchema,
  type InstanceSettings,
  type OAuthProviderFormData,
  type OptionalProvider,
} from "@/models/User";

// Per-provider copy only; the flow (save both, disable by clearing both, redirect URI to register) is identical.
const providerCopy: Record<
  OptionalProvider,
  { label: string; console: string; idPlaceholder: string; secretPlaceholder: string }
> = {
  google: {
    label: "Google",
    console: "Create an OAuth client (Web application) in Google Cloud Console",
    idPlaceholder: "1234567890-abc.apps.googleusercontent.com",
    secretPlaceholder: "GOCSPX-…",
  },
  discord: {
    label: "Discord",
    console: "Create an application in the Discord Developer Portal (OAuth2 → General)",
    idPlaceholder: "123456789012345678",
    secretPlaceholder: "Client secret from the OAuth2 page",
  },
};

interface OAuthProviderSectionProps {
  provider: OptionalProvider;
  settings: InstanceSettings;
}

// Optional sign-in providers (ADR 0040) for non-GitHub users; owner-only, secret is write-only.
export const OAuthProviderSection = ({ provider, settings }: OAuthProviderSectionProps) => {
  const copy = providerCopy[provider];
  const update = useUpdateProviderOAuth(provider, copy.label);
  const configured = settings[`${provider}_oauth_configured`] ?? false;
  const callback = settings[`${provider}_oauth_callback`] ?? "";
  const form = useForm<OAuthProviderFormData>({
    defaultValues: { client_id: settings[`${provider}_oauth_client_id`] ?? "", client_secret: "" },
    resolver: zodResolver(OAuthProviderFormSchema),
  });

  const save = async (data: OAuthProviderFormData) => {
    try {
      await update.mutateAsync(data);
      form.reset({ client_id: data.client_id, client_secret: "" });
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  return (
    <SettingsCard
      id={`${provider}-sign-in`}
      title={`${copy.label} sign-in`}
      description={
        <>
          Lets people without a GitHub account — clients, stakeholders — sign in with {copy.label}. {copy.console} and
          register <code className="font-mono text-xs">{callback}</code> as its redirect URI. Allowlist their email
          under Instance access.
          {configured && " Currently enabled."}
        </>
      }
    >
      <form onSubmit={form.handleSubmit(save)} className="space-y-4">
        <FormInput
          control={form.control}
          name="client_id"
          id={`${provider}_client_id`}
          label="Client ID"
          placeholder={copy.idPlaceholder}
          autoComplete="off"
          className="w-full sm:w-96"
        />
        <FormInput
          control={form.control}
          name="client_secret"
          id={`${provider}_client_secret`}
          label="Client secret"
          type="password"
          placeholder={configured ? "Stored — enter a new one to rotate" : copy.secretPlaceholder}
          autoComplete="new-password"
          className="w-full sm:w-96"
        />
        <div className="flex flex-wrap gap-2">
          <Button type="submit" disabled={form.formState.isSubmitting}>
            {form.formState.isSubmitting ? "Saving…" : "Save"}
          </Button>
          {configured && (
            <Button
              type="button"
              variant="outline"
              disabled={update.isPending}
              onClick={() => save({ client_id: "", client_secret: "" })}
            >
              Disable {copy.label} sign-in
            </Button>
          )}
        </div>
      </form>
    </SettingsCard>
  );
};
