import { Suspense } from "react";

import { LazyCreateDocForm } from "@/components/doc/LazyCreateDocForm";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveDocFormSchema, type SaveDocFormData } from "@/models/Doc";
import { emptyDocForm } from "@/utils/emptyDocJson";

// Opens the New doc dialog in projectId, landing in folderId or the project's default folder; undefined when the
// viewer may not create docs, so callers hide the button.
export const useCreateDocDialog = (projectId: string, folderId?: string): (() => void) | undefined => {
  const canCreate = useAreaAccess()?.("newDoc") ?? false;
  const { open } = useFormDialog();
  if (!canCreate) return undefined;
  return () =>
    void open<SaveDocFormData>({
      title: "New doc",
      schema: SaveDocFormSchema,
      okLabel: "Create",
      header: <ProjectDialogHeader title="New doc" />,
      form: (
        <Suspense fallback={<LoadingDisplay />}>
          <LazyCreateDocForm defaultProjectId={projectId} />
        </Suspense>
      ),
      formOptions: { defaultValues: { ...emptyDocForm(), project_id: projectId, folder_id: folderId } },
    });
};
