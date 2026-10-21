import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { useState } from "react";
import { Controller, useForm } from "react-hook-form";

import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { AutomationSdkSetup } from "@/components/automation/AutomationSdkSetup";
import { PermissionLevels } from "@/components/access/PermissionLevels";
import { FormInput } from "@/components/FormInput";
import { useCreateAutomation, useFetchScopeCatalog } from "@/hooks/AutomationHooks";
import { CreateAutomationFormSchema, type CreateAutomationFormData } from "@/models/Automation";

// A host runs the automation on credentials of its own, so the token minted with it isn't shown; the Token tab can mint one.
export const NewAutomationDialog = () => {
  const [open, setOpen] = useState(false);
  const create = useCreateAutomation();
  const { data: scopeCatalog = [] } = useFetchScopeCatalog();

  const form = useForm<CreateAutomationFormData>({
    defaultValues: { name: "", scopes: [] },
    resolver: zodResolver(CreateAutomationFormSchema),
  });

  const onSubmit = async (data: CreateAutomationFormData) => {
    try {
      await create.mutateAsync({ name: data.name, scopes: data.scopes });
      form.reset();
    } catch {
      // Failure toast is handled by useCreateAutomation; the form stays open to retry.
    }
  };

  const onOpenChange = (next: boolean) => {
    setOpen(next);
    if (next) return;
    create.reset();
    form.reset();
  };

  const minted = create.data;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <ActionButton permission="automations:write" verb="create automations" icon={Plus} type="button">
          New automation
        </ActionButton>
      </DialogTrigger>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>New automation</DialogTitle>
          <DialogDescription>Name it and pick what its token may touch.</DialogDescription>
        </DialogHeader>
        {!minted && (
          <DialogBody>
            <form id="new-automation" onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormInput control={form.control} name="name" label="Name" placeholder="e.g. Slack notifier" />
              <Controller
                control={form.control}
                name="scopes"
                render={({ field, fieldState }) => (
                  <fieldset className="@container space-y-2">
                    <legend className="text-sm font-medium">Scopes</legend>
                    <PermissionLevels entries={scopeCatalog} value={field.value} onChange={field.onChange} />
                    {fieldState.error && (
                      <p role="alert" className="text-sm text-destructive">
                        {fieldState.error.message}
                      </p>
                    )}
                  </fieldset>
                )}
              />
            </form>
          </DialogBody>
        )}
        {!minted && (
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" form="new-automation" loading={form.formState.isSubmitting}>
              Create automation
            </Button>
          </DialogFooter>
        )}
        {minted && (
          <DialogBody className="space-y-4">
            <p className="text-sm font-medium">{minted.automation.name} created</p>
            <AutomationSdkSetup automationId={minted.automation.id} />
          </DialogBody>
        )}
        {minted && (
          <DialogFooter>
            <Button type="button" onClick={() => onOpenChange(false)}>
              Done
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
};
