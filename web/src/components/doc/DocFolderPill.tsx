import { ChevronRightIcon } from "lucide-react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { PillPicker } from "@/components/PillPicker";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import type { SaveDocFormData } from "@/models/Doc";

// The New doc header's folder pill; a folder from another project reads as the picked project's default.
export const DocFolderPill = () => {
  const { watch, setValue } = useFormDialogContext<SaveDocFormData>();
  const { data: folders } = useFetchDocFolders(watch("project_id"));
  const folderId = watch("folder_id");
  if (!folders || folders.length === 0) return null;
  const folder = folders.find((f) => f.id === folderId) ?? folders.find((f) => f.is_default);

  return (
    <>
      <PillPicker label={folder?.name ?? "Folder"} items={folders} onPick={(id) => setValue("folder_id", id)} />
      <ChevronRightIcon className="size-3.5 text-muted-foreground" aria-hidden />
    </>
  );
};
