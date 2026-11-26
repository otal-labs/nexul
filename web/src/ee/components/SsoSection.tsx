import { ArrowRightLeft, Building2, Trash2 } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsHint } from "@/components/settings/SettingsHint";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { SamlProviderDialog } from "@/ee/components/SamlProviderDialog";
import { SamlRows } from "@/ee/components/SamlRows";
import { SsoOidcRows } from "@/ee/components/SsoOidcRows";
import { SsoProviderDialog } from "@/ee/components/SsoProviderDialog";
import { useRemoveSaml, useRemoveSso, useSamlSettings, useSsoSettings } from "@/ee/hooks/SsoHooks";
import { samlSaved, type SamlSettings } from "@/ee/models/Saml";
import type { SsoSettings } from "@/ee/models/Sso";
import { MOVE_SIGN_INS_TAB } from "@/models/Access";

// The one provider an instance holds, by protocol; none until either is saved.
type Saved = "oidc" | "saml" | "none";

const savedOf = (sso: SsoSettings, saml: SamlSettings): Saved => {
  if (samlSaved(saml)) return "saml";
  if (sso.configured) return "oidc";
  return "none";
};

const statusWord = (sso: SsoSettings, configured: boolean): string => {
  if (!sso.unlocked) return "Needs a license";
  if (!configured) return "Off";
  return "On";
};

const SsoStatus = ({ sso, configured }: { sso: SsoSettings; configured: boolean }) => (
  <SettingsStatus tone={sso.unlocked && configured ? "success" : "muted"}>{statusWord(sso, configured)}</SettingsStatus>
);

const RemoveProviderButton = ({ saved }: { saved: Saved }) => {
  const removeOidc = useRemoveSso();
  const removeSaml = useRemoveSaml();
  const remove = saved === "saml" ? removeSaml : removeOidc;
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

const otherProtocol: Record<Exclude<Saved, "none">, string> = { oidc: "SAML", saml: "OpenID Connect" };

// After a switch of protocol, the people the old one signed in are moved onto their accounts from their own page.
const MoveSignInsRow = () => (
  <SettingsRows>
    <SettingsRow label="Move sign-ins" description="Link the people the other protocol signed in to the accounts their first sign-in here made, or expect their sign-in.">
      <ActionButton permission="sign_ins:move" verb="move sign-ins" variant="outline" size="sm" icon={ArrowRightLeft} to={`/settings/sign-in/${MOVE_SIGN_INS_TAB}`}>
        Move sign-ins
      </ActionButton>
    </SettingsRow>
  </SettingsRows>
);

const SsoBody = ({ sso, saml, saved }: { sso: SsoSettings; saml: SamlSettings; saved: Saved }) => (
  <div className="space-y-4">
    {!sso.unlocked && (
      <InlineNotice title="Single sign-on needs a license key" tone="warning">
        The sign-in button is off until a key that unlocks it is added under Instance.
        {saved !== "none" && " The saved settings stay and work again with one."}
      </InlineNotice>
    )}
    {sso.unlocked && saved === "none" && (
      <EmptyRow flush>People sign in with GitHub, Google, Discord or a passkey. Set up a provider over OpenID Connect, or SAML if it speaks nothing else.</EmptyRow>
    )}
    {saved === "saml" && <SamlRows saml={saml} />}
    {saved !== "saml" && (saved === "oidc" || sso.unlocked) && <SsoOidcRows sso={sso} />}
    {saved !== "none" && <MoveSignInsRow />}
    {saved !== "none" && <SettingsHint>An instance holds one provider: remove this one to set up {otherProtocol[saved]} instead.</SettingsHint>}
  </div>
);

const SsoFooter = ({ sso, saml, saved }: { sso: SsoSettings; saml: SamlSettings; saved: Saved }) => (
  <>
    {saved !== "none" && <RemoveProviderButton saved={saved} />}
    {sso.unlocked && (
      <span className="ml-auto flex gap-2">
        {saved !== "oidc" && <SamlProviderDialog saved={saml} />}
        {saved !== "saml" && <SsoProviderDialog saved={sso} />}
      </span>
    )}
  </>
);

export const SsoSection = () => {
  const oidc = useSsoSettings();
  const saml = useSamlSettings();
  const error = oidc.error ?? saml.error;
  const ready = oidc.data && saml.data && { sso: oidc.data, saml: saml.data, saved: savedOf(oidc.data, saml.data) };
  return (
    <SettingsCard
      id="single-sign-on"
      title="Single sign-on"
      description="Lets people sign in through your company's identity provider, over OpenID Connect or SAML."
      icon={Building2}
      aside={ready && <SsoStatus sso={ready.sso} configured={ready.sso.configured || ready.saml.configured} />}
      footer={ready && <SsoFooter {...ready} />}
    >
      {(oidc.isPending || saml.isPending) && !error && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {ready && <SsoBody {...ready} />}
    </SettingsCard>
  );
};
