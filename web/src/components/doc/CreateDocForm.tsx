import { useEffect } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { FormInput } from "@/components/FormInput";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { FormSelect } from "@/components/ticket/FormSelect";
import { useCreateDoc } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { SaveDocFormData } from "@/models/Doc";

interface CreateDocFormProps {
  defaultProjectId?: string;
}

export const CreateDocForm = ({ defaultProjectId = "" }: CreateDocFormProps) => {
  const { control, setValue, watch, onSubmit, setLoading } = useFormDialogContext<SaveDocFormData>();
  const createDoc = useCreateDoc();
  const { data: projects } = useFetchProjects();

  const ready = projects != null;
  const noProjects = ready && (projects?.length ?? 0) === 0;

  useEffect(() => {
    setLoading(!ready || noProjects);
  }, [ready, noProjects, setLoading]);

  // Seed once reference data loads; the dialog mounts fresh per open (react-confirm), same pattern as CreateTicketForm.
  useEffect(() => {
    if (!ready) return;
    setValue("project_id", (defaultProjectId || projects?.[0]?.id) ?? "");
  }, [ready, projects, defaultProjectId, setValue]);

  onSubmit(async (input) => {
    const doc = await createDoc.mutateAsync(input);
    return { id: doc.id, ...input };
  });

  const projectOptions = (projects ?? []).map((project) => ({ value: project.id, label: project.name }));

  return (
    <div className="space-y-4">
      {noProjects && (
        <NoDataDisplay message="Create a project first — every doc belongs to exactly one project." />
      )}
      {!noProjects && (
        <>
          <FormSelect control={control} name="project_id" label="Project" options={projectOptions} />
          <FormInput control={control} name="title" label="Title" placeholder="Doc title" />
          <div>
            <span className="text-sm font-medium">Body</span>
            <RichTextEditor value={watch("body")} onChange={(value) => setValue("body", value)} aria-label="Body" />
          </div>
        </>
      )}
    </div>
  );
};
