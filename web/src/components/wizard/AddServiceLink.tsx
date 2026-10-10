import { PlusIcon } from "lucide-react";
import { Link } from "react-router";

import { Button, type ButtonProps } from "@/components/ui/button";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { NEW_PROJECT_PATH } from "@/models/Project";

interface AddServiceLinkProps extends Omit<ButtonProps, "asChild" | "children"> {
  // Preselects the project on entry (spec §1, door 2: "project page's Add service; Topology empty state").
  projectId?: string;
  children?: React.ReactNode;
}

// Drop-in door-2 trigger: opens the project wizard at the repository step, project preselected when known.
// Exported for pages this ticket doesn't own (Topology's empty state) to render without duplicating the route.
export const AddServiceLink = ({ projectId, children, variant, size, className, ...props }: AddServiceLinkProps) => {
  const canAdd = useAreaAccess()?.("newProject") ?? false;
const wsPath = useWorkspacePath();
  if (!canAdd) return null;
  return (
    <Button asChild variant={variant} size={size} className={className} {...props}>
      <Link to={wsPath(projectId ? `/wizard/project/repository?project=${projectId}&add=1` : NEW_PROJECT_PATH)}>
        <PlusIcon className="size-3.5" aria-hidden />
        {children ?? "Add service"}
      </Link>
    </Button>
  );
};
