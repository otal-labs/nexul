import { RefreshCw } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { SamlCertificateRows } from "@/ee/components/SamlCertificateRows";
import { CopyValue } from "@/components/settings/CopyValue";
import { SsoJoinRole } from "@/ee/components/SsoJoinRole";
import { SsoOnlyRow } from "@/ee/components/SsoOnlyRow";
import { useSaveSaml, useSetSamlOnly } from "@/ee/hooks/SsoHooks";
import type { SamlSettings } from "@/ee/models/Saml";
import { formatRelativeTime } from "@/utils/TimeUtility";

// Where the metadata comes from, and why its last read failed while the certificates read before it stay in use.
const MetadataOrigin = ({ saml }: { saml: SamlSettings }) => (
  <>
    <span className="block break-all">{saml.metadata_url || "Uploaded XML, replaced by uploading again."}</span>
    {saml.metadata_refresh_error !== "" && <span className="block text-destructive">{saml.metadata_refresh_error}</span>}
  </>
);

// Whether the metadata URL's last read worked, and Refresh to read it now.
const MetadataRefresh = ({ saml }: { saml: SamlSettings }) => {
  const save = useSaveSaml();
  return (
    <span className="flex min-w-0 items-center justify-between gap-2">
      {saml.metadata_refresh_error !== "" && <SettingsStatus tone="destructive">Refresh failing</SettingsStatus>}
      {saml.metadata_refresh_error === "" && (
        <SettingsStatus tone="success" detail={saml.metadata_refreshed_at && formatRelativeTime(saml.metadata_refreshed_at)}>
          Read
        </SettingsStatus>
      )}
      <ActionButton
        permission="instance:write"
        verb="refresh the metadata"
        variant="ghost"
        size="sm"
        icon={RefreshCw}
        loading={save.isPending}
        onClick={() => save.mutate({ source: "url", metadata_url: saml.metadata_url })}
      >
        Refresh
      </ActionButton>
    </span>
  );
};

// The SAML provider's rows: what to register at the provider, what was read from its metadata, and what is saved.
export const SamlRows = ({ saml }: { saml: SamlSettings }) => {
  const setOnly = useSetSamlOnly();
  return (
    <SettingsRows>
      <SettingsRow label="Entity ID" description="Register Nexul at your provider with these, or give it the metadata link.">
        <CopyValue value={saml.sp.entity_id} label="Copy entity ID" />
      </SettingsRow>
      <SettingsRow label="Assertion consumer service URL">
        <CopyValue value={saml.sp.acs_url} label="Copy assertion consumer service URL" />
      </SettingsRow>
      <SettingsRow label="Metadata" description={<a href={saml.sp.metadata_url} className="text-brand underline-offset-2 hover:underline">Open Nexul's metadata XML</a>}>
        <CopyValue value={saml.sp.metadata_url} label="Copy metadata link" />
      </SettingsRow>
      <SettingsRow label="Provider" description={saml.idp_entity_id}>
        <span className="truncate text-sm">{saml.display_name}</span>
      </SettingsRow>
      <SettingsRow label="Provider metadata" description={<MetadataOrigin saml={saml} />}>
        {saml.metadata_url === "" && <SettingsStatus tone="muted">Uploaded</SettingsStatus>}
        {saml.metadata_url !== "" && <MetadataRefresh saml={saml} />}
      </SettingsRow>
      <SettingsRow
        label="Email attribute"
        description="Signed by your provider, this address is trusted as theirs: if people can edit it there, they can sign in as someone else here."
      >
        <code className="truncate font-mono text-xs" title={saml.email_attribute}>
          {saml.email_attribute}
        </code>
      </SettingsRow>
      {saml.name_attribute !== "" && (
        <SettingsRow label="Name attribute">
          <code className="truncate font-mono text-xs">{saml.name_attribute}</code>
        </SettingsRow>
      )}
      <SettingsRow label="New people join as" description="Their first sign-in makes an account in the default workspace with this role.">
        <SsoJoinRole roleId={saml.join_role_id} />
      </SettingsRow>
      {saml.unlocked && <SsoOnlyRow only={saml.only} setOnly={setOnly} />}
      <SamlCertificateRows saml={saml} />
    </SettingsRows>
  );
};
