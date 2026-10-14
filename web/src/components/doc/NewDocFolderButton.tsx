import { FolderPlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { DocFolderNameForm } from "@/components/doc/DocFolderNameForm";
import { useFormDialog } from "@/hooks/useFormDialog";
import { DocFolderFormSchema, type DocFolderFormData } from "@/models/DocFolder";

interface NewDocFolderButtonProps {
  projectId: string;
}

export const NewDocFolderButton = ({ projectId }: NewDocFolderButtonProps) => {
  const { open } = useFormDialog();
  return (
    <ActionButton
      permission="docs:write"
      projectId={projectId}
      verb="create folders"
      icon={FolderPlusIcon}
      variant="ghost"
      size="icon"
      className="size-8"
      aria-label="New folder"
      title="New folder"
      onClick={() =>
        void open<DocFolderFormData>({
          title: "New folder",
          schema: DocFolderFormSchema,
          okLabel: "Create folder",
          form: <DocFolderNameForm projectId={projectId} />,
          formOptions: { defaultValues: { name: "" } },
        })
      }
    />
  );
};
