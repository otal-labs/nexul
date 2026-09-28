import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { errorMessage } from "@/api/client";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useSetConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import {
  connectorAppConfigFormSchema,
  type AppConfigStatus,
  type ConnectorAppConfigFormData,
} from "@/models/Connectors";

interface ConnectorAppConfigFormProps {
  connectorId: string;
  onSaved?: () => void;
  // A registered app to edit: its fields start filled, and a blank secret keeps the stored one.
  current?: AppConfigStatus;
}

// Owner-only OAuth app registration (CN3a); shared by the Settings card and the connector card's "Set up app" dialog.
export const ConnectorAppConfigForm = ({ connectorId, onSaved, current }: ConnectorAppConfigFormProps) => {
  const setAppConfig = useSetConnectorAppConfig();
  const githubApp = connectorId === "github";
  const editing = current?.configured ?? false;

  const form = useForm<ConnectorAppConfigFormData>({
    defaultValues: {
      client_id: current?.client_id ?? "",
      client_secret: "",
      base_url: current?.base_url ?? "",
      app_slug: current?.app_slug ?? "",
    },
    resolver: zodResolver(connectorAppConfigFormSchema(githubApp, !editing)),
  });

  const onSubmit = async (data: ConnectorAppConfigFormData) => {
    try {
      await setAppConfig.mutateAsync({ id: connectorId, ...data });
      form.reset(data);
      onSaved?.();
    } catch (err) {
      form.setError("root", { message: errorMessage(err) });
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
      <FormInput
        control={form.control}
        name="client_id"
        id={`${connectorId}-app-client-id`}
        label="Client ID"
        placeholder={githubApp ? "Iv1.abc123" : ""}
      />
      <FormInput
        control={form.control}
        name="client_secret"
        id={`${connectorId}-app-client-secret`}
        label="Client secret"
        type="password"
        placeholder={editing ? "Leave blank to keep the current secret" : "••••••••••••••••"}
      />
      {githubApp && (
        <FormInput
          control={form.control}
          name="base_url"
          id={`${connectorId}-app-base-url`}
          label="Base URL (optional)"
          placeholder="https://github.example.com (leave blank for github.com)"
        />
      )}
      {githubApp && (
        <FormInput
          control={form.control}
          name="app_slug"
          id={`${connectorId}-app-slug`}
          label="App slug"
          placeholder="your-app-slug"
        />
      )}
      {form.formState.errors.root && (
        <p role="alert" className="text-sm text-destructive">
          {form.formState.errors.root.message}
        </p>
      )}
      <Button type="submit" disabled={form.formState.isSubmitting}>
        {form.formState.isSubmitting ? "Saving…" : "Save"}
      </Button>
    </form>
  );
};
