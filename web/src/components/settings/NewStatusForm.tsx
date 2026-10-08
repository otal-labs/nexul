import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { PlusIcon } from "lucide-react";

import { FormInput } from "@/components/FormInput";
import { StatusIconPicker } from "@/components/settings/StatusIconPicker";
import { Button } from "@/components/ui/button";
import { useCreateStatus } from "@/hooks/StatusHooks";
import { SaveStatusFormSchema, type SaveStatusFormData, type StatusKind } from "@/models/Status";

interface NewStatusFormProps {
  projectId: string;
  kind: StatusKind;
  onDone: () => void;
}

export const NewStatusForm = ({ projectId, kind, onDone }: NewStatusFormProps) => {
  const createStatus = useCreateStatus();
  const form = useForm<SaveStatusFormData>({
    defaultValues: { name: "", kind, icon: "" },
    resolver: zodResolver(SaveStatusFormSchema),
  });

  return (
    <form
      className="mt-2 flex flex-wrap items-end gap-2"
      onSubmit={form.handleSubmit(async (data) => {
        await createStatus.mutateAsync({ ...data, project_id: projectId });
        onDone();
      })}
    >
      <FormInput
        control={form.control}
        name="name"
        id="new-status-name"
        label="New status name"
        placeholder="New status name"
        hideLabel
        className="w-full sm:w-48"
      />
      <Controller
        control={form.control}
        name="icon"
        render={({ field }) => <StatusIconPicker label="New status icon" value={field.value} onChange={field.onChange} />}
      />
      <Button type="submit" size="icon" variant="outline" aria-label="Add column" loading={form.formState.isSubmitting}>
        <PlusIcon className="size-4" />
      </Button>
    </form>
  );
};
