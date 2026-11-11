import { Building2, Trash2 } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { CopyButton } from "@/components/settings/CopyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { SsoOnlyRow } from "@/ee/components/SsoOnlyRow";
import { SsoProviderDialog } from "@/ee/components/SsoProviderDialog";
import { useRemoveSso, useSsoSettings } from "@/ee/hooks/SsoHooks";
import type { SsoSettings } from "@/ee/models/Sso";
import { useFetchWorkspaceRoles } from "@/hooks/RoleHooks";
import { INSTANCE_WORKSPACE_ID } from "@/models/Workspace";

const SsoStatus = ({ sso }: { sso: SsoSettings }) => {
  if (!sso.unlocked) return <SettingsStatus tone="muted">Needs a license</SettingsStatus>;
  if (!sso.configured) return <SettingsStatus tone="muted">Off</SettingsStatus>;
  return <SettingsStatus tone="success">On</SettingsStatus>;
};

const RedirectUri = ({ uri }: { uri: string }) => (
  <span className="flex w-full min-w-0 items-center gap-1 rounded-md bg-surface-2 py-0.5 pr-0.5 pl-2.5 ring-1 ring-border">
    <code className="min-w-0 flex-1 truncate font-mono text-xs" title={uri}>
      {uri}
    </code>
    <CopyButton value={uri} label="Copy redirect URI" iconOnly variant="ghost" />
  </span>
);

const JoinRole = ({ roleId }: { roleId: string }) => {
  const { data: roles } = useFetchWorkspaceRoles(INSTANCE_WORKSPACE_ID);
  return <span className="truncate text-sm">{roles?.find((r) => r.id === roleId)?.name ?? "A role that no longer exists"}</span>;
};

const SsoRows = ({ sso }: { sso: SsoSettings }) => (
  <SettingsRows>
    <SettingsRow label="Redirect URI" description="Register this as the redirect URI for this instance at your provider.">
      <RedirectUri uri={sso.redirect_uri} />
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
          <JoinRole roleId={sso.join_role_id} />
        </SettingsRow>
        {sso.unlocked && <SsoOnlyRow only={sso.only} />}
      </>
    )}
  </SettingsRows>
);

const RemoveSsoButton = () => {
  const remove = useRemoveSso();
  return (
    <ConfirmDestroyButton
      icon={Trash2}
      idleLabel="Remove provider"
      confirmLabel="Remove"
      labelled
      loading={remove.isPending}
      onConfirm={() => remove.mutate()}
      gate={{ permission: "instance:write", verb: "remove single sign-on" }}
    />
  );
};

const SsoBody = ({ sso }: { sso: SsoSettings }) => (
  <div className="space-y-4">
    {!sso.unlocked && (
      <InlineNotice title="Single sign-on needs a license key" tone="warning">
        The sign-in button is off until a key that unlocks it is added under Instance.
        {sso.configured && " The saved settings stay and work again with one."}
      </InlineNotice>
    )}
    {sso.unlocked && !sso.configured && <EmptyRow flush>People sign in with GitHub, Google, Discord or a passkey.</EmptyRow>}
    {(sso.configured || sso.unlocked) && <SsoRows sso={sso} />}
  </div>
);

export const SsoSection = () => {
  const { data, isPending, error } = useSsoSettings();
  return (
    <SettingsCard
      id="single-sign-on"
      title="Single sign-on"
      description="Lets people sign in through your company's identity provider, over OpenID Connect."
      icon={Building2}
      aside={data && <SsoStatus sso={data} />}
      footer={
        data && (
          <>
            {data.configured && <RemoveSsoButton />}
            {data.unlocked && (
              <span className="ml-auto">
                <SsoProviderDialog saved={data} />
              </span>
            )}
          </>
        )
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <SsoBody sso={data} />}
    </SettingsCard>
  );
};
