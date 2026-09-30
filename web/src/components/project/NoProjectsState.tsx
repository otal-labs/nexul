import { FolderPlusIcon } from "lucide-react";
import { Link } from "react-router";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { NEW_PROJECT_PATH } from "@/models/Project";

interface NoProjectsStateProps {
  message: string;
}

// A workspace starts with no project; every page that needs one points at the project wizard instead of erroring.
export const NoProjectsState = ({ message }: NoProjectsStateProps) => {
  const canCreate = useAreaAccess()?.("newProject") ?? false;
  const wsPath = useWorkspacePath();
  return (
    <EmptyState
      icon={FolderPlusIcon}
      title="No projects yet"
      message={message}
      action={
        canCreate && (
          <Button asChild size="sm">
            <Link to={wsPath(NEW_PROJECT_PATH)}>New project</Link>
          </Button>
        )
      }
    />
  );
};
