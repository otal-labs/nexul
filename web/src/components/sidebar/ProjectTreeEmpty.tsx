import { PlusIcon } from "lucide-react";
import { Link } from "react-router";

import { navLinkClass } from "@/components/SidebarNav";
import { NEW_PROJECT_PATH } from "@/models/Project";

// The expanded tree's empty state: the hover-only + would hide the one action a new workspace needs.
export const ProjectTreeEmpty = () => (
  <div className="flex flex-col gap-0.5">
    <p className="px-2.5 py-1 text-xs text-muted-foreground">No projects yet.</p>
    <Link to={NEW_PROJECT_PATH} className={navLinkClass({ isActive: false })}>
      <PlusIcon className="size-4 shrink-0" aria-hidden />
      <span>Create your first project</span>
    </Link>
  </div>
);
