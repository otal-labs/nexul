import { useState } from "react";
import { ScrollText, Trash2 } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { GatedSaveBar } from "@/components/settings/GatedSaveBar";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { AuditDaysRow, AuditDestinationRow, AuditSecretRow, AuditSendRow } from "@/ee/components/AuditLogFields";
import { AuditDeliveriesRow, AuditNewSecretRow, RotateSecretButton } from "@/ee/components/AuditStreamRows";
import { useAuditForm } from "@/ee/components/useAuditForm";
import { useAuditSettings, useRemoveAuditSettings } from "@/ee/hooks/AuditLogHooks";
import { AuditMinDays, type AuditSettings } from "@/ee/models/AuditLog";

const FORM_ID = "audit-log-form";
const TITLE = "Audit log";
const DESCRIPTION = "Keep the record of changes longer than 45 days, and send each change, signed, to your own log store as it happens.";

const AuditStatus = ({ settings }: { settings: AuditSettings }) => (
  <SettingsStatus tone={settings.streaming ? "success" : "muted"} detail={`kept ${settings.retention_days} days`}>
    {settings.streaming ? "Streaming" : "Not streaming"}
  </SettingsStatus>
);

const isSaved = (s: AuditSettings) => s.retention_days !== AuditMinDays || s.stream_url !== "";

const RemoveButton = () => {
  const remove = useRemoveAuditSettings();
  return (
    <ConfirmDestroyButton
      icon={Trash2}
      idleLabel="Remove settings"
      confirmLabel="Remove"
      labelled
      loading={remove.isPending}
      onConfirm={() => remove.mutate()}
      gate={{ permission: "instance:write", verb: "remove the audit log settings" }}
    />
  );
};

const LockedNotice = ({ settings }: { settings: AuditSettings }) => (
  <InlineNotice title="Keeping the log longer and streaming it need a license key" tone="warning" className="mb-4">
    {isSaved(settings) && `Rows are still kept ${settings.retention_days} days and queued deliveries wait; nothing is sent until `}
    {!isSaved(settings) && "Rows are kept 45 days until "}
    <a href="#license" className="text-brand underline-offset-4 hover:underline">
      License
    </a>{" "}
    holds a key that includes Audit log. Remove the settings to go back to 45 days.
  </InlineNotice>
);

const AuditLogBody = ({ settings }: { settings: AuditSettings }) => {
  const { form, disabled, saved, madeSecret, onSubmit } = useAuditForm(settings);
  const [rotated, setRotated] = useState<string>();
  const secret = rotated ?? madeSecret;
  return (
    <SettingsCard
      id="audit-log"
      title={TITLE}
      description={DESCRIPTION}
      icon={ScrollText}
      aside={<AuditStatus settings={settings} />}
      footer={
        <>
          {isSaved(settings) && <RemoveButton />}
          {settings.stream_secret_set && <RotateSecretButton unlocked={settings.unlocked} onRotated={setRotated} />}
          <GatedSaveBar
            permission="instance:write"
            verb="change the audit log settings"
            form={FORM_ID}
            dirty={form.formState.isDirty}
            saving={form.formState.isSubmitting}
            saved={saved}
            onDiscard={() => form.reset()}
          />
        </>
      }
    >
      {!settings.unlocked && <LockedNotice settings={settings} />}
      <form id={FORM_ID} onSubmit={onSubmit}>
        <SettingsRows>
          <AuditDaysRow control={form.control} disabled={disabled} />
          <AuditDestinationRow control={form.control} disabled={disabled} />
          <AuditSecretRow control={form.control} disabled={disabled} secretSet={settings.stream_secret_set} />
          {secret && <AuditNewSecretRow secret={secret} />}
          <AuditSendRow control={form.control} disabled={disabled} />
          {settings.stream_url !== "" && <AuditDeliveriesRow settings={settings} />}
        </SettingsRows>
      </form>
    </SettingsCard>
  );
};

export const AuditLogSection = () => {
  const { data, isPending, error } = useAuditSettings();
  return (
    <>
      {isPending && (
        <SettingsCard id="audit-log" title={TITLE} icon={ScrollText}>
          <LoadingDisplay />
        </SettingsCard>
      )}
      {error && (
        <SettingsCard id="audit-log" title={TITLE} icon={ScrollText}>
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {data && <AuditLogBody settings={data} />}
    </>
  );
};
