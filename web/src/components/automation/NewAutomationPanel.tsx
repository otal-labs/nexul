import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { AutomationTokenReveal } from "@/components/automation/AutomationTokenReveal";
import { PermissionGrid } from "@/components/access/PermissionGrid";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useCreateAutomation, useFetchScopeCatalog } from "@/hooks/AutomationHooks";
import { CreateAutomationFormSchema, type CreateAutomationFormData } from "@/models/Automation";
import type { AutomationTokenMint } from "@/models/Automation";

// The minted token shows exactly once with the SDK commands, never again after this panel closes.
export const NewAutomationPanel = () => {
  const [open, setOpen] = useState(false);
  const [minted, setMinted] = useState<AutomationTokenMint | null>(null);
  const create = useCreateAutomation();
  const { data: scopeCatalog = [] } = useFetchScopeCatalog();

  const form = useForm<CreateAutomationFormData>({
    defaultValues: { name: "", scopes: [] },
    resolver: zodResolver(CreateAutomationFormSchema),
  });

  const onSubmit = async (data: CreateAutomationFormData) => {
    try {
      const result = await create.mutateAsync({ name: data.name, scopes: data.scopes });
      setMinted(result);
      form.reset();
    } catch {
      // Failure toast is handled by useCreateAutomation; the form stays open to retry.
    }
  };

  const close = () => {
    setOpen(false);
    setMinted(null);
  };

  return (
    <div className="space-y-4">
      {!open && <Button onClick={() => setOpen(true)}>New automation</Button>}
      {open && !minted && (
        <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-4 rounded-lg border border-border bg-card p-4 duration-200 ease-out">
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormInput control={form.control} name="name" label="Name" placeholder="e.g. Slack notifier" />
            <Controller
              control={form.control}
              name="scopes"
              render={({ field, fieldState }) => (
                <fieldset className="space-y-2">
                  <legend className="text-sm font-medium">Scopes</legend>
                  <PermissionGrid entries={scopeCatalog} value={field.value} onChange={field.onChange} />
                  {fieldState.error && (
                    <p role="alert" className="text-sm text-destructive">
                      {fieldState.error.message}
                    </p>
                  )}
                </fieldset>
              )}
            />
            <div className="flex flex-wrap gap-2">
              <Button type="submit" disabled={form.formState.isSubmitting}>
                {form.formState.isSubmitting ? "Creating…" : "Create automation"}
              </Button>
              <Button type="button" variant="ghost" onClick={close}>
                Cancel
              </Button>
            </div>
          </form>
        </div>
      )}
      {minted && (
        <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-4 rounded-lg border border-border bg-card p-4 duration-200 ease-out">
          <p className="text-sm font-medium">{minted.automation.name} created</p>
          <AutomationTokenReveal token={minted.token} />
          <Button type="button" variant="outline" size="sm" onClick={close}>
            Done
          </Button>
        </div>
      )}
    </div>
  );
};
