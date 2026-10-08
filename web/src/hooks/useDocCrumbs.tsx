import type { Crumb } from "@/components/PageBreadcrumb";
import { useFetchDocFolders } from "@/hooks/DocFolderHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Doc } from "@/models/Doc";

// Docs, then the folder the doc lives in once the folders load.
export const useDocCrumbs = (doc: Doc): Crumb[] => {
  const wsPath = useWorkspacePath();
  const { data: folders } = useFetchDocFolders(doc.project_id);
  const folder = folders?.find((f) => f.id === doc.folder_id);
  const docs = { label: "Docs", to: wsPath("/docs") };
  return folder ? [docs, { label: folder.name }] : [docs];
};
