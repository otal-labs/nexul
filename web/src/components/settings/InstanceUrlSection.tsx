import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Check, Copy } from "lucide-react";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useUpdateSettings } from "@/hooks/AuthHooks";
import { InstanceURLFormSchema, type InstanceURLFormData, type InstanceSettings } from "@/models/User";

interface InstanceUrlSectionProps {
  settings: InstanceSettings;
}

export const InstanceUrlSection = ({ settings }: InstanceUrlSectionProps) => {
  const updateSettings = useUpdateSettings();
  const [copied, setCopied] = useState(false);

  const form = useForm<InstanceURLFormData>({
    defaultValues: { instance_url: settings.instance_url },
    resolver: zodResolver(InstanceURLFormSchema),
  });

  const onSubmit = async (data: InstanceURLFormData) => {
    try {
      await updateSettings.mutateAsync(data.instance_url);
      form.reset(data);
    } catch {
      // Error is surfaced by the hook's toast; the form stays open to retry.
    }
  };

  const copyCallback = async () => {
    try {
      await navigator.clipboard.writeText(settings.oauth_callback);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard may be unavailable; the chip stays visible for manual copy.
    }
  };

  return (
    <SettingsCard
      id="instance"
      title="Instance"
      description={
        <>
          The address this instance is reached at. Changing it regenerates connection tokens.
          Register the callback below as the GitHub OAuth callback.
        </>
      }
    >
      <form
        key={settings.settings_version}
        onSubmit={form.handleSubmit(onSubmit)}
        className="space-y-4"
      >
        <div className="flex flex-wrap items-end gap-2">
          <FormInput
            control={form.control}
            name="instance_url"
            id="instance_url"
            label="Instance URL"
            placeholder="https://deploy.example.com"
            className="w-full sm:w-96"
          />
          <Button type="submit" disabled={form.formState.isSubmitting}>
            {form.formState.isSubmitting ? "Saving…" : "Save"}
          </Button>
        </div>

        {/* Mono chip with copy affordance instead of a bare <code> block. */}
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="shrink-0 text-xs text-muted-foreground">OAuth callback</span>
          <span className="flex min-w-0 items-center gap-1.5 rounded-md border border-border bg-muted px-2 py-1">
            <code className="min-w-0 flex-1 font-mono text-xs break-all text-foreground">
              {settings.oauth_callback}
            </code>
            <button
              type="button"
              aria-label="Copy OAuth callback"
              title="Copy"
              onClick={copyCallback}
              className="shrink-0 rounded-sm text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
            >
              {copied && <Check className="size-3.5 text-success" />}
              {!copied && <Copy className="size-3.5" />}
            </button>
          </span>
        </div>
      </form>
    </SettingsCard>
  );
};
