import { FolderPlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DocFolderNameForm } from "@/components/doc/DocFolderNameForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { DocFolderFormSchema, type DocFolderFormData } from "@/models/DocFolder";

interface NewDocFolderButtonProps {
  projectId: string;
}

export const NewDocFolderButton = ({ projectId }: NewDocFolderButtonProps) => {
  const canWrite = useHasPermission("docs:write");
  const { open } = useFormDialog();
  if (!canWrite) return null;
  return (
    <Button
      variant="ghost"
      size="icon"
      className="size-8"
      aria-label="New folder"
      title="New folder"
      onClick={() =>
        void open<DocFolderFormData>({
          title: "New folder",
          schema: DocFolderFormSchema,
          okLabel: "Create",
          form: <DocFolderNameForm projectId={projectId} />,
          formOptions: { defaultValues: { name: "" } },
        })
      }
    >
      <FolderPlusIcon className="size-4" aria-hidden />
    </Button>
  );
};
