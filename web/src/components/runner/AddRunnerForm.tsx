import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { RunnerEnrollmentFormSchema, type RunnerEnrollmentFormData } from "@/models/Runner";
import { cn } from "@/lib/utils";

interface AddRunnerFormProps {
  // Set when adding to a known machine: the runner joins that machine's pool.
  machineName?: string | undefined;
  pending: boolean;
  onSubmit: (input: RunnerEnrollmentFormData) => Promise<unknown>;
}

// A failed submit is toasted by the enrollment hook, so the form only keeps its values for another try.
export const AddRunnerForm = ({ machineName, pending, onSubmit }: AddRunnerFormProps) => {
  const form = useForm<RunnerEnrollmentFormData>({
    resolver: zodResolver(RunnerEnrollmentFormSchema),
    defaultValues: { name: "", machine: machineName ?? "", gitToken: "" },
  });
  const submit = form.handleSubmit((input) => onSubmit(input).catch(() => undefined));

  return (
    <form onSubmit={submit} className="space-y-4">
      <FormInput control={form.control} name="name" label="Runner name" placeholder="e.g. build-box-1" autoFocus />
      <FormInput
        control={form.control}
        name="machine"
        label="Machine"
        placeholder="Defaults to the runner's name"
        readOnly={!!machineName}
        className={cn(machineName && "text-muted-foreground")}
      />
      <div className="space-y-2">
        <FormInput
          control={form.control}
          name="gitToken"
          label="GitHub token"
          type="password"
          autoComplete="off"
          placeholder="ghp_..."
        />
        <p className="text-xs text-muted-foreground">
          Optional. Used to clone private repositories; it goes into the command, never to this instance.
        </p>
      </div>
      <Button type="submit" loading={pending}>
        Create install command
      </Button>
    </form>
  );
};
