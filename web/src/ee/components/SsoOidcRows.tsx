import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SsoCopyValue } from "@/ee/components/SsoCopyValue";
import { SsoJoinRole } from "@/ee/components/SsoJoinRole";
import { SsoOnlyRow } from "@/ee/components/SsoOnlyRow";
import { useSetSsoOnly } from "@/ee/hooks/SsoHooks";
import type { SsoSettings } from "@/ee/models/Sso";

// The OpenID Connect provider's rows: the redirect URI to register, then what is saved.
export const SsoOidcRows = ({ sso }: { sso: SsoSettings }) => {
  const setOnly = useSetSsoOnly();
  return (
    <SettingsRows>
      <SettingsRow label="Redirect URI" description="Register this as the redirect URI for this instance at your provider.">
        <SsoCopyValue value={sso.redirect_uri} label="Copy redirect URI" />
      </SettingsRow>
      {sso.configured && (
        <>
          <SettingsRow label="Provider" description={sso.issuer_url}>
            <span className="truncate text-sm">{sso.display_name}</span>
          </SettingsRow>
          <SettingsRow label="Client ID">
            <code className="truncate font-mono text-xs" title={sso.client_id}>
              {sso.client_id}
            </code>
          </SettingsRow>
          <SettingsRow label="New people join as" description="Their first sign-in makes an account in the default workspace with this role.">
            <SsoJoinRole roleId={sso.join_role_id} />
          </SettingsRow>
          {sso.unlocked && <SsoOnlyRow only={sso.only} setOnly={setOnly} />}
        </>
      )}
    </SettingsRows>
  );
};
