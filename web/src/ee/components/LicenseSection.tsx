import { KeyRound, Trash2 } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Fact } from "@/components/Fact";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { LicenseKeyDialog } from "@/ee/components/LicenseKeyDialog";
import { useLicense, useRemoveLicense } from "@/ee/hooks/LicenseHooks";
import { featureDescription, featureLabel, formatExpiry, LicenseState, type License } from "@/ee/models/License";
import { daysUntil } from "@/utils/TimeUtility";

const statusOf = (license: License): { headline: string; tone: SettingsStatusTone } => {
  if (license.state === LicenseState.Active) return { headline: "Active", tone: "success" };
  if (license.state === LicenseState.Expiring) {
    return { headline: `Expires in ${daysUntil(license.expires_at ?? "")} days`, tone: "warning" };
  }
  if (license.state === LicenseState.Expired) return { headline: "Expired", tone: "destructive" };
  if (license.state === LicenseState.Invalid) return { headline: "Not valid here", tone: "destructive" };
  return { headline: "No key", tone: "muted" };
};

const LicenseStatus = () => {
  const { data } = useLicense();
  if (!data) return null;
  const { headline, tone } = statusOf(data);
  return <SettingsStatus tone={tone}>{headline}</SettingsStatus>;
};

const LicenseFacts = ({ license }: { license: License }) => (
  <dl className="grid grid-cols-2 gap-x-6 gap-y-4">
    <Fact label="Licensee">{license.licensee}</Fact>
    <Fact label="Expires">{license.expires_at && formatExpiry(license.expires_at)}</Fact>
    <div className="col-span-2">
      <Fact label="Instance">
        <span className="font-mono break-all">{license.instance_url}</span>
      </Fact>
    </div>
  </dl>
);

const FeatureRow = ({ feature, on }: { feature: string; on: boolean }) => (
  <SettingsRow label={featureLabel(feature)} description={featureDescription(feature)}>
    <SettingsStatus tone={on ? "success" : "muted"}>{on ? "Unlocked" : "Off"}</SettingsStatus>
  </SettingsRow>
);

// What this key unlocks, one row per feature, so a feature's own card says the same thing as its row here.
const LicenseFeatures = ({ license }: { license: License }) => (
  <SettingsRows>
    {license.features.map((feature) => (
      <FeatureRow key={feature} feature={feature} on={license.state === LicenseState.Active || license.state === LicenseState.Expiring} />
    ))}
  </SettingsRows>
);

const LicenseBody = ({ license }: { license: License }) => (
  <div className="space-y-4">
    {license.state === LicenseState.None && <EmptyRow flush>This instance runs without enterprise features.</EmptyRow>}
    {license.licensee !== "" && <LicenseFacts license={license} />}
    {license.licensee !== "" && license.features.length > 0 && <LicenseFeatures license={license} />}
    {license.state === LicenseState.Expired && (
      <InlineNotice title="The enterprise features are off" tone="warning">
        Everything they made is kept. Paste a renewed key to turn them back on.
      </InlineNotice>
    )}
    {license.problem && <InlineNotice title="This key unlocks nothing here">{license.problem}</InlineNotice>}
  </div>
);

const RemoveLicenseButton = () => {
  const remove = useRemoveLicense();
  return (
    <ConfirmDestroyButton
      icon={Trash2}
      idleLabel="Remove key"
      confirmLabel="Remove"
      labelled
      loading={remove.isPending}
      onConfirm={() => remove.mutate()}
      gate={{ permission: "instance:write", verb: "remove the license key" }}
    />
  );
};

const LicenseFooter = ({ license }: { license: License }) => {
  const held = license.state !== LicenseState.None;
  return (
    <>
      {held && <RemoveLicenseButton />}
      <LicenseKeyDialog replacing={held} />
    </>
  );
};

export const LicenseSection = () => {
  const { data, isPending, error } = useLicense();

  return (
    <SettingsCard
      id="license"
      title="License"
      description="A license key unlocks the enterprise features on this instance. It's checked here, without contacting anyone."
      icon={KeyRound}
      aside={<LicenseStatus />}
      footer={data && <LicenseFooter license={data} />}
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <LicenseBody license={data} />}
    </SettingsCard>
  );
};
