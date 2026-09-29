import { CheckCircle2 } from "lucide-react";

import { Fact } from "@/components/Fact";
import { OAuthProviderActions } from "@/components/settings/OAuthProviderActions";
import { OAuthProviderForm } from "@/components/settings/OAuthProviderForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { oauthProviderCopy, type InstanceSettings, type OptionalProvider } from "@/models/User";

interface OAuthProviderSectionProps {
  provider: OptionalProvider;
  settings: InstanceSettings;
}

// Optional sign-in providers (ADR 0040) for non-GitHub users; instance:write to change, secret is write-only.
export const OAuthProviderSection = ({ provider, settings }: OAuthProviderSectionProps) => {
  const copy = oauthProviderCopy[provider];
  const enabled = settings[`${provider}_oauth_configured`] ?? false;
  const callback = settings[`${provider}_oauth_callback`] ?? "";
  const clientId = settings[`${provider}_oauth_client_id`] ?? "";

  return (
    <SettingsCard
      id={`${provider}-sign-in`}
      title={`${copy.label} sign-in`}
      description={
        <>
          Lets people without a GitHub account — clients, stakeholders — sign in with {copy.label} through an
          invitation link. {copy.console} and register <code className="font-mono text-xs">{callback}</code> as its
          redirect URI.
        </>
      }
      footer={enabled && <OAuthProviderActions provider={provider} clientId={clientId} />}
    >
      {!enabled && <OAuthProviderForm provider={provider} />}
      {enabled && (
        <div className="space-y-4">
          <p className="flex items-center gap-2 text-sm">
            <CheckCircle2 className="size-4 text-success" aria-hidden />
            Enabled
          </p>
          <dl className="grid gap-4 sm:grid-cols-2">
            <Fact label="Client ID">
              <span className="truncate font-mono text-xs">{clientId}</span>
            </Fact>
            <Fact label="Redirect URI">
              <span className="truncate font-mono text-xs">{callback}</span>
            </Fact>
          </dl>
        </div>
      )}
    </SettingsCard>
  );
};
