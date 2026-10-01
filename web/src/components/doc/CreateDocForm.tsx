import { useEffect } from "react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { dialogTitleInputClass } from "@/components/ticket/ticketFormPillStyles";
import { useCreateDoc } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import type { SaveDocFormData } from "@/models/Doc";

interface CreateDocFormProps {
  defaultProjectId?: string;
}

// The create-ticket dialog's shape: ProjectDialogHeader picks the project, then a borderless title over the body.
export const CreateDocForm = ({ defaultProjectId = "" }: CreateDocFormProps) => {
  const { register, formState, getValues, setValue, watch, onSubmit, setLoading, submit } =
    useFormDialogContext<SaveDocFormData>();
  const createDoc = useCreateDoc();
  const { data: projects } = useFetchProjects();

  const ready = projects != null;
  const noProjects = ready && projects.length === 0;

  useEffect(() => {
    setLoading(!ready || noProjects);
  }, [ready, noProjects, setLoading]);

  // Seeds only an empty pick, so a refetch never undoes the header's choice.
  useEffect(() => {
    if (!ready || getValues("project_id")) return;
    setValue("project_id", (defaultProjectId || projects[0]?.id) ?? "");
  }, [ready, projects, defaultProjectId, getValues, setValue]);

  // The folder belongs to the project the dialog opened in; picking another project files the doc in that one's default.
  onSubmit(async ({ folder_id, ...input }) => {
    const doc = await createDoc.mutateAsync(input.project_id === defaultProjectId ? { ...input, folder_id } : input);
    return { id: doc.id, ...input };
  });

  const titleError = formState.errors.title?.message;

  return (
    <div
      className="space-y-3"
      // Capture: the body editor would otherwise take Mod-Enter as a line break before the form sees it.
      onKeyDownCapture={(e) => {
        if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
          e.preventDefault();
          e.stopPropagation();
          submit();
        }
      }}
    >
      {noProjects && <NoDataDisplay message="Create a project first — every doc belongs to exactly one project." />}
      {!noProjects && (
        <div className="space-y-3">
          <div>
            <input
              {...register("title")}
              autoFocus
              aria-label="Title"
              aria-invalid={titleError != null}
              placeholder="Doc title"
              className={dialogTitleInputClass}
            />
            {titleError && (
              <p role="alert" className="mt-1 text-sm text-destructive">
                {titleError}
              </p>
            )}
          </div>
          <div className="max-h-[40dvh] overflow-y-auto">
            <RichTextEditor compact value={watch("body")} onChange={(value) => setValue("body", value)} aria-label="Body" />
          </div>
        </div>
      )}
    </div>
  );
};
