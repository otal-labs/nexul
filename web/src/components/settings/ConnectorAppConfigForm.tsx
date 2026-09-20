import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { errorMessage } from "@/api/client";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useSetConnectorAppConfig } from "@/hooks/ConnectorsHooks";
import { connectorAppConfigFormSchema, type ConnectorAppConfigFormData } from "@/models/Connectors";

interface ConnectorAppConfigFormProps {
  connectorId: string;
  onSaved?: () => void;
}

// Owner-only OAuth app registration (CN3a); shared by the Settings card and the connector card's "Set up app" dialog.
export const ConnectorAppConfigForm = ({ connectorId, onSaved }: ConnectorAppConfigFormProps) => {
  const setAppConfig = useSetConnectorAppConfig();
  const githubApp = connectorId === "github";

  const form = useForm<ConnectorAppConfigFormData>({
    defaultValues: { client_id: "", client_secret: "", base_url: "", app_slug: "" },
    resolver: zodResolver(connectorAppConfigFormSchema(githubApp)),
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
        placeholder="••••••••••••••••"
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
