import { PlusIcon } from "lucide-react";
import { Link } from "react-router";

import { Button, type ButtonProps } from "@/components/ui/button";

interface AddServiceLinkProps extends Omit<ButtonProps, "asChild" | "children"> {
  // Preselects the project on entry (spec §1, door 2: "project page's Add service; Topology empty state").
  projectId?: string;
  children?: React.ReactNode;
}

// Drop-in door-2 trigger: opens the project wizard at the repository step, project preselected when known.
// Exported for pages this ticket doesn't own (Topology's empty state) to render without duplicating the route.
export const AddServiceLink = ({ projectId, children, variant, size, className, ...props }: AddServiceLinkProps) => (
  <Button asChild variant={variant} size={size} className={className} {...props}>
    <Link to={projectId ? `/wizard/project/repository?project=${projectId}` : "/wizard/project/project"}>
      <PlusIcon className="size-3.5" aria-hidden />
      {children ?? "Add service"}
    </Link>
  </Button>
);
