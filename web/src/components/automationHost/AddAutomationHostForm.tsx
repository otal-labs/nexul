import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import {
  AutomationHostEnrollmentFormSchema,
  type AutomationHostEnrollmentFormData,
} from "@/models/AutomationHost";

interface AddAutomationHostFormProps {
  pending: boolean;
  onSubmit: (input: AutomationHostEnrollmentFormData) => Promise<unknown>;
}

// A failed submit is toasted by the enrollment hook, so the form only keeps its values for another try.
export const AddAutomationHostForm = ({ pending, onSubmit }: AddAutomationHostFormProps) => {
  const form = useForm<AutomationHostEnrollmentFormData>({
    resolver: zodResolver(AutomationHostEnrollmentFormSchema),
    defaultValues: { name: "", machine: "" },
  });
  const submit = form.handleSubmit((input) => onSubmit(input).catch(() => undefined));

  return (
    <form onSubmit={submit} className="space-y-4">
      <FormInput control={form.control} name="name" label="Host name" placeholder="e.g. jobs-1" autoFocus />
      <FormInput control={form.control} name="machine" label="Machine" placeholder="Defaults to the computer it runs on" />
      <Button type="submit" loading={pending}>
        Create install command
      </Button>
    </form>
  );
};
