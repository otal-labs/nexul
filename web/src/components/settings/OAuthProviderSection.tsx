import { CopyButton } from "@/components/settings/CopyButton";
import { OAuthProviderActions } from "@/components/settings/OAuthProviderActions";
import { OAuthProviderForm } from "@/components/settings/OAuthProviderForm";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
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
      description={`Lets invited people without GitHub, such as clients, sign in with ${copy.label}.`}
      aside={<SettingsStatus tone={enabled ? "success" : "muted"}>{enabled ? "Enabled" : "Off"}</SettingsStatus>}
      footer={enabled && <OAuthProviderActions provider={provider} clientId={clientId} />}
    >
      <SettingsRows>
        <SettingsRow label="Redirect URI" description={`${copy.console}, and register this as its redirect URI.`}>
          <span className="flex w-full min-w-0 items-center gap-1 rounded-md bg-surface-2 py-0.5 pr-0.5 pl-2.5 ring-1 ring-border">
            <code className="min-w-0 flex-1 truncate font-mono text-xs" title={callback}>
              {callback}
            </code>
            <CopyButton value={callback} label="Copy redirect URI" iconOnly variant="ghost" />
          </span>
        </SettingsRow>
        {enabled && (
          <SettingsRow label="Client ID">
            <code className="truncate font-mono text-xs" title={clientId}>
              {clientId}
            </code>
          </SettingsRow>
        )}
      </SettingsRows>
      {!enabled && (
        <div className="mt-5">
          <OAuthProviderForm provider={provider} />
        </div>
      )}
    </SettingsCard>
  );
};
