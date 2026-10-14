import { FolderPlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { EmptyState } from "@/components/EmptyState";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { wizardDoorPermissions } from "@/models/Access";
import { NEW_PROJECT_PATH } from "@/models/Project";

interface NoProjectsStateProps {
  message: string;
}

// A workspace starts with no project; every page that needs one points at the project wizard instead of erroring.
export const NoProjectsState = ({ message }: NoProjectsStateProps) => {
  const wsPath = useWorkspacePath();
  return (
    <EmptyState
      icon={FolderPlusIcon}
      title="No projects yet"
      message={message}
      action={
        <ActionButton permission={wizardDoorPermissions(true)} verb="create projects" size="sm" to={wsPath(NEW_PROJECT_PATH)}>
          New project
        </ActionButton>
      }
    />
  );
};
