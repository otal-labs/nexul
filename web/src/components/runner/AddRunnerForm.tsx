import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { RunnerEnrollmentFormSchema, type RunnerEnrollmentFormData } from "@/models/Runner";

interface AddRunnerFormProps {
  pending: boolean;
  onSubmit: (input: RunnerEnrollmentFormData) => Promise<unknown>;
}

// A failed submit is toasted by the enrollment hook, so the form only keeps its values for another try.
export const AddRunnerForm = ({ pending, onSubmit }: AddRunnerFormProps) => {
  const form = useForm<RunnerEnrollmentFormData>({
    resolver: zodResolver(RunnerEnrollmentFormSchema),
    defaultValues: { name: "" },
  });
  const submit = form.handleSubmit((input) => onSubmit(input).catch(() => undefined));

  return (
    <form onSubmit={submit} className="space-y-4">
      <FormInput control={form.control} name="name" label="Runner name" placeholder="e.g. build-box-1" autoFocus />
      <DialogFooter>
        <Button type="submit" loading={pending}>
          Create install command
        </Button>
      </DialogFooter>
    </form>
  );
};
