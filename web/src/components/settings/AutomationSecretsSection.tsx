import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { AutomationSecretRow } from "@/components/settings/AutomationSecretRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useFetchAutomationSecrets, useSetAutomationSecret } from "@/hooks/AutomationSecretHooks";
import {
  SaveAutomationSecretFormSchema,
  type SaveAutomationSecretFormData,
} from "@/models/AutomationSecret";

// Shared workspace pool (ADR 0047); saving an existing name replaces its value, the backend has no partial update.
export const AutomationSecretsSection = () => {
  const { data, isPending, error } = useFetchAutomationSecrets();
  const setSecret = useSetAutomationSecret();

  const form = useForm<SaveAutomationSecretFormData>({
    defaultValues: { name: "", value: "" },
    resolver: zodResolver(SaveAutomationSecretFormSchema),
  });

  const onSubmit = async (data: SaveAutomationSecretFormData) => {
    await setSecret.mutateAsync(data);
    form.reset();
  };

  return (
    <SettingsCard
      id="automation-secrets"
      title="Automation secrets"
      description="Shared with every automation as ctx.secrets.NAME. Values are write-only — once saved, they can never be read back, only replaced or deleted."
    >
      <div className="space-y-6">
        <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-wrap items-end gap-2">
          <FormInput
            control={form.control}
            name="name"
            id="secret-name"
            label="Name"
            placeholder="e.g. SLACK_WEBHOOK_URL"
            className="w-full font-mono sm:w-64"
          />
          <FormInput
            control={form.control}
            name="value"
            id="secret-value"
            label="Value"
            type="password"
            className="w-full sm:w-64"
          />
          <Button type="submit" disabled={form.formState.isSubmitting}>
            {form.formState.isSubmitting ? "Saving…" : "Save secret"}
          </Button>
        </form>

        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {data && data.length === 0 && <NoDataDisplay message="No secrets yet" />}
        {data && data.length > 0 && (
          <ul className="divide-y divide-border overflow-hidden rounded-md border">
            {data.map((secret) => (
              <AutomationSecretRow key={secret.name} secret={secret} />
            ))}
          </ul>
        )}
      </div>
    </SettingsCard>
  );
};
