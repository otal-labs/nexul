import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { CopyButton } from "@/components/settings/CopyButton";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { useUpdateSettings } from "@/hooks/AuthHooks";
import { useFlash } from "@/hooks/useFlash";
import { InstanceURLFormSchema, type InstanceURLFormData, type InstanceSettings } from "@/models/User";

const FORM_ID = "instance-url-form";

interface InstanceUrlSectionProps {
  settings: InstanceSettings;
}

export const InstanceUrlSection = ({ settings }: InstanceUrlSectionProps) => {
  const updateSettings = useUpdateSettings();
  const [saved, flash] = useFlash();

  const form = useForm<InstanceURLFormData>({
    defaultValues: { instance_url: settings.instance_url },
    resolver: zodResolver(InstanceURLFormSchema),
  });

  const onSubmit = async (data: InstanceURLFormData) => {
    try {
      await updateSettings.mutateAsync(data.instance_url);
      form.reset(data);
      flash();
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  return (
    <SettingsCard
      id="instance"
      title="Address"
      footer={
        <SettingsSaveBar
          form={FORM_ID}
          dirty={form.formState.isDirty}
          saving={form.formState.isSubmitting}
          saved={saved}
          onDiscard={() => form.reset()}
        />
      }
    >
      <form id={FORM_ID} key={settings.settings_version} onSubmit={form.handleSubmit(onSubmit)}>
        <SettingsRows>
          <SettingsRow
            label="Instance URL"
            description="Where people reach this instance. Changing it issues new connection tokens."
            htmlFor="instance_url"
          >
            <div className="w-full">
              <FormInput
                control={form.control}
                name="instance_url"
                id="instance_url"
                label="Instance URL"
                hideLabel
                placeholder="https://deploy.example.com"
              />
            </div>
          </SettingsRow>
          <SettingsRow label="OAuth callback" description="Register it as the callback of the GitHub OAuth app.">
            <span className="flex w-full min-w-0 items-center gap-1 rounded-md bg-surface-2 py-0.5 pr-0.5 pl-2.5 ring-1 ring-border">
              <code className="min-w-0 flex-1 truncate font-mono text-xs" title={settings.oauth_callback}>
                {settings.oauth_callback}
              </code>
              <CopyButton value={settings.oauth_callback} label="Copy OAuth callback" iconOnly variant="ghost" />
            </span>
          </SettingsRow>
        </SettingsRows>
      </form>
    </SettingsCard>
  );
};
