import { useNavigate } from "react-router";

import { useOpenCreateDoc } from "@/hooks/useCreateDocDialog";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { docPath, projectToken, type Project } from "@/models/Project";

// Opens the New doc dialog and, once it creates one, the new doc.
export const useNewDoc = (project: Project) => {
  const createDoc = useOpenCreateDoc(project.id);
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  return async () => {
    const id = await createDoc();
    if (id) void navigate(wsPath(docPath(projectToken(project), id)));
  };
};
