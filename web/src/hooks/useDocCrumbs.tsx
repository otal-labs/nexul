import type { Crumb } from "@/components/PageBreadcrumb";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Doc } from "@/models/Doc";
import { folderChain } from "@/utils/DocFolderTreeUtility";

// Docs, then the folder the doc lives in and those it sits inside, once the folders load.
export const useDocCrumbs = (doc: Doc): Crumb[] => {
  const wsPath = useWorkspacePath();
  const { data: folders } = useFetchDocFolders(doc.project_id);
  const docs = { label: "Docs", to: wsPath("/docs") };
  return [docs, ...folderChain(folders ?? [], doc.folder_id).map((f) => ({ label: f.name }))];
};
