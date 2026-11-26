import { useState } from "react";

import { ActionButton } from "@/components/action/ActionButton";
import { CopyValue } from "@/components/settings/CopyValue";
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
// An off provider keeps its form folded behind Set up, so the page is not mostly an unrequested form.
export const OAuthProviderSection = ({ provider, settings }: OAuthProviderSectionProps) => {
  const [settingUp, setSettingUp] = useState(false);
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
      footer={
        <>
          {enabled && <OAuthProviderActions provider={provider} clientId={clientId} />}
          {!enabled && (
            <ActionButton
              permission="instance:write"
              verb="change sign-in providers"
              variant={settingUp ? "ghost" : "outline"}
              size="sm"
              aria-expanded={settingUp}
              onClick={() => setSettingUp(!settingUp)}
            >
              {settingUp ? "Cancel" : "Set up"}
            </ActionButton>
          )}
        </>
      }
    >
      <SettingsRows>
        <SettingsRow label="Redirect URI" description={`${copy.console}, and register this as its redirect URI.`} stacked>
          <CopyValue value={callback} label="Copy redirect URI" />
        </SettingsRow>
        {enabled && (
          <SettingsRow label="Client ID" stacked>
            <CopyValue value={clientId} label="Copy client ID" />
          </SettingsRow>
        )}
      </SettingsRows>
      {!enabled && settingUp && (
        <div className="mt-5">
          <OAuthProviderForm provider={provider} />
        </div>
      )}
    </SettingsCard>
  );
};
