import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useSetAutomationSecret } from "@/hooks/AutomationSecretHooks";
import { SaveAutomationSecretFormSchema, type SaveAutomationSecretFormData } from "@/models/AutomationSecret";

interface AutomationSecretFormProps {
  // Replacing an existing secret fixes its name; a new one types it.
  name?: string;
  onDone: () => void;
}

// Saving an existing name replaces its value; the backend has no partial update (ADR 0047).
export const AutomationSecretForm = ({ name, onDone }: AutomationSecretFormProps) => {
  const setSecret = useSetAutomationSecret();
  const form = useForm<SaveAutomationSecretFormData>({
    defaultValues: { name: name ?? "", value: "" },
    resolver: zodResolver(SaveAutomationSecretFormSchema),
  });

  const onSubmit = async (data: SaveAutomationSecretFormData) => {
    await setSecret.mutateAsync(data);
    form.reset();
    onDone();
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="settle-in flex flex-wrap items-end gap-2 rounded-md bg-surface-2 p-3">
      <FormInput
        control={form.control}
        name="name"
        id="secret-name"
        label="Name"
        placeholder="e.g. SLACK_WEBHOOK_URL"
        className="w-full font-mono sm:w-64"
        readOnly={!!name}
        autoFocus={!name}
        autoComplete="off"
        data-1p-ignore
      />
      <FormInput
        control={form.control}
        name="value"
        id="secret-value"
        label={name ? "New value" : "Value"}
        type="password"
        className="w-full sm:w-64"
        autoFocus={!!name}
        autoComplete="off"
        data-1p-ignore
      />
      <div className="flex gap-1">
        <Button type="submit" loading={form.formState.isSubmitting}>
          {name ? "Replace value" : "Save secret"}
        </Button>
        <Button type="button" variant="ghost" onClick={onDone}>
          Cancel
        </Button>
      </div>
    </form>
  );
};
