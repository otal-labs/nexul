import { PlusIcon } from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import type { ButtonProps } from "@/components/ui/button";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { wizardDoorPermissions } from "@/models/Access";
import { NEW_PROJECT_PATH } from "@/models/Project";

interface AddServiceLinkProps extends Pick<ButtonProps, "variant" | "size" | "className" | "children"> {
  // Preselects the project on entry (spec §1, door 2: "project page's Add service; Topology empty state").
  projectId?: string;
}

// Drop-in door-2 trigger: opens the project wizard at the repository step, project preselected when known.
export const AddServiceLink = ({ projectId, children, ...props }: AddServiceLinkProps) => {
  const wsPath = useWorkspacePath();
  return (
    <ActionButton
      permission={wizardDoorPermissions(!projectId)}
      projectId={projectId}
      verb="add a service"
      icon={PlusIcon}
      to={wsPath(projectId ? `/wizard/project/repository?project=${projectId}&add=1` : NEW_PROJECT_PATH)}
      {...props}
    >
      {children ?? "Add service"}
    </ActionButton>
  );
};
