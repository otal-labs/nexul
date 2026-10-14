import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormCombobox } from "@/components/FormCombobox";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import {
  AutomationHostEnrollmentFormSchema,
  type AutomationHostEnrollmentFormData,
} from "@/models/AutomationHost";

interface AddAutomationHostFormProps {
  pending: boolean;
  onSubmit: (input: AutomationHostEnrollmentFormData) => Promise<unknown>;
  // The machines the viewer Manages, for anyone but the instance Owner: they must pick one. The Owner types a name, or none.
  managedMachines?: string[] | undefined;
}

// A failed submit is toasted by the enrollment hook, so the form only keeps its values for another try.
export const AddAutomationHostForm = ({ pending, onSubmit, managedMachines }: AddAutomationHostFormProps) => {
  const form = useForm<AutomationHostEnrollmentFormData>({
    resolver: zodResolver(AutomationHostEnrollmentFormSchema),
    defaultValues: { name: "", machine: "" },
  });
  const submit = form.handleSubmit((input) => {
    if (managedMachines && !input.machine) return form.setError("machine", { message: "Choose a machine you manage" });
    return onSubmit(input).catch(() => undefined);
  });

  return (
    <form onSubmit={submit} className="space-y-4">
      <FormInput control={form.control} name="name" label="Host name" placeholder="e.g. jobs-1" autoFocus />
      {!managedMachines && (
        <FormInput control={form.control} name="machine" label="Machine" placeholder="Defaults to the computer it runs on" />
      )}
      {managedMachines && (
        <FormCombobox
          control={form.control}
          name="machine"
          label="Machine"
          options={managedMachines.map((name) => ({ value: name, label: name }))}
          placeholder="Choose a machine…"
        />
      )}
      <Button type="submit" loading={pending}>
        Create install command
      </Button>
    </form>
  );
};
