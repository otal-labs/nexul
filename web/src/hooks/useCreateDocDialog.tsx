import { Suspense } from "react";

import { DocFolderPill } from "@/components/doc/DocFolderPill";
import { LazyCreateDocForm } from "@/components/doc/LazyCreateDocForm";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ProjectDialogHeader } from "@/components/project/ProjectDialogHeader";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFormDialog } from "@/hooks/useFormDialog";
import { SaveDocFormSchema, type SaveDocFormData } from "@/models/Doc";
import { emptyDocForm } from "@/utils/emptyDocJson";

// Opens the New doc dialog in projectId, landing in folderId or the project's default folder, and resolves with the
// new doc's id (undefined when cancelled). An action control gates it itself.
export const useOpenCreateDoc = (projectId: string, folderId?: string): (() => Promise<string | undefined>) => {
  const { open } = useFormDialog();
  return async () => {
    const result = await open<SaveDocFormData>({
      title: "New doc",
      schema: SaveDocFormSchema,
      okLabel: "Create doc",
      header: (
        <ProjectDialogHeader title="New doc">
          <DocFolderPill />
        </ProjectDialogHeader>
      ),
      form: (
        <Suspense fallback={<LoadingDisplay />}>
          <LazyCreateDocForm defaultProjectId={projectId} />
        </Suspense>
      ),
      formOptions: { defaultValues: { ...emptyDocForm(), project_id: projectId, folder_id: folderId } },
    });
    return (result.data as { id?: string } | null)?.id;
  };
};

// The same, undefined when the viewer may not create docs, for a command that is left out instead of locked.
export const useCreateDocDialog = (projectId: string, folderId?: string): (() => Promise<string | undefined>) | undefined => {
  const canCreate = useAreaAccess()?.("newDoc") ?? false;
  const open = useOpenCreateDoc(projectId, folderId);
  return canCreate ? open : undefined;
};
