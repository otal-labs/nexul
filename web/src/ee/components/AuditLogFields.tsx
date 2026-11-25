import { Send } from "lucide-react";
import { Controller, useWatch, type Control } from "react-hook-form";

import { ActionButton } from "@/components/action/ActionButton";
import { FormInput } from "@/components/FormInput";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Switch } from "@/components/ui/switch";
import { useTestAuditStream } from "@/ee/hooks/AuditLogHooks";
import { AuditMaxDays, AuditMinDays, type AuditFormData } from "@/ee/models/AuditLog";

type FieldProps = { control: Control<AuditFormData>; disabled: boolean };

export const AuditDaysRow = ({ control, disabled }: FieldProps) => (
  <SettingsRow
    label="Days kept"
    description={`From ${AuditMinDays} to ${AuditMaxDays.toLocaleString("en")}. Each day the rows older than this are deleted, with their deliveries.`}
    htmlFor="audit-days"
  >
    <div className="w-full">
      <FormInput control={control} name="retention_days" id="audit-days" label="Days kept" hideLabel inputMode="numeric" disabled={disabled} />
    </div>
  </SettingsRow>
);

const TestButton = ({ control, disabled }: FieldProps) => {
  const test = useTestAuditStream();
  const [url, secret] = useWatch({ control, name: ["stream_url", "stream_secret"] });
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
      <ActionButton
        permission="instance:write"
        verb="test the audit stream"
        type="button"
        variant="outline"
        size="sm"
        icon={Send}
        loading={test.isPending}
        disabled={disabled || url.trim() === ""}
        onClick={() => test.mutate({ stream_url: url.trim(), ...(secret.trim() !== "" && { stream_secret: secret.trim() }) })}
      >
        Send a test
      </ActionButton>
      {test.data?.error && (
        <SettingsStatus tone="destructive" detail={test.data.error}>
          Not taken
        </SettingsStatus>
      )}
      {test.data && !test.data.error && (
        <SettingsStatus tone="success" detail={`answered ${test.data.status}`}>
          Received
        </SettingsStatus>
      )}
    </div>
  );
};

export const AuditDestinationRow = ({ control, disabled }: FieldProps) => (
  <SettingsRow
    label="Destination"
    description="Each change is posted here as JSON, signed. An https address; http only on this machine, for trying a receiver out."
    htmlFor="audit-url"
    stacked
  >
    <div className="w-full space-y-2">
      <FormInput control={control} name="stream_url" id="audit-url" label="Destination" hideLabel type="url" placeholder="https://logs.example.com/nexul" disabled={disabled} />
      <TestButton control={control} disabled={disabled} />
    </div>
  </SettingsRow>
);

export const AuditSecretRow = ({ control, disabled, secretSet }: FieldProps & { secretSet: boolean }) => (
  <SettingsRow
    label="Signing secret"
    description="X-Nexul-Signature carries the HMAC-SHA256 of each body under it. A secret reference such as op://vault/item/field is read from your secrets manager."
    htmlFor="audit-secret"
  >
    <div className="w-full">
      <FormInput
        control={control}
        name="stream_secret"
        id="audit-secret"
        label="Signing secret"
        hideLabel
        type="password"
        autoComplete="off"
        placeholder={secretSet ? "Saved; type to replace it" : "Leave blank to have one made"}
        disabled={disabled}
      />
    </div>
  </SettingsRow>
);

export const AuditSendRow = ({ control, disabled }: FieldProps) => (
  <SettingsRow label="Send each change" description="Changes made while this is off are kept here and never sent." htmlFor="audit-send">
    <Controller
      control={control}
      name="stream_enabled"
      render={({ field, fieldState }) => (
        <div className="flex flex-col items-end gap-1">
          <Switch id="audit-send" checked={field.value} onCheckedChange={field.onChange} disabled={disabled} aria-invalid={!!fieldState.error} />
          {fieldState.error && (
            <p role="alert" className="text-xs text-destructive">
              {fieldState.error.message}
            </p>
          )}
        </div>
      )}
    />
  </SettingsRow>
);
