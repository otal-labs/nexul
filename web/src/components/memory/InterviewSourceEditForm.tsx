import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { InterviewSourceTextFields } from "@/components/memory/InterviewSourceTextFields";
import { useEditTextSource } from "@/hooks/InterviewSourceHooks";
import { EditTextSourceFormSchema, type EditTextSourceFormData, type InterviewSource } from "@/models/InterviewSource";

interface InterviewSourceEditFormProps {
  source: InterviewSource;
  onClose: () => void;
}

// Editing pasted text in place of its row: the same label and text fields the add form shows.
export const InterviewSourceEditForm = ({ source, onClose }: InterviewSourceEditFormProps) => {
  const form = useForm<EditTextSourceFormData>({
    resolver: zodResolver(EditTextSourceFormSchema),
    defaultValues: { label: source.label, body: source.body ?? "" },
  });
  const edit = useEditTextSource(source.project_id);
  const submit = form.handleSubmit((values) => edit.mutate({ id: source.id, ...values }, { onSuccess: onClose }));

  return (
    <FormProvider {...form}>
      <form onSubmit={submit} aria-label={`Edit ${source.label}`} className="my-2 space-y-4 rounded-lg border border-border p-4">
        <InterviewSourceTextFields />
        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" size="sm" loading={edit.isPending}>
            Save
          </Button>
        </div>
      </form>
    </FormProvider>
  );
};
