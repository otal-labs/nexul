import type { ReactNode } from "react";
import { ChevronLeft, CrownIcon } from "lucide-react";
import { Link } from "react-router";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";

interface RoleDetailHeaderProps {
  title: string;
  line: string;
  owner?: boolean;
  actions?: ReactNode;
}

// The open pane's title over one muted line; below 1024px, where the list and the pane take turns, a link back to the list.
export const RoleDetailHeader = ({ title, line, owner = false, actions }: RoleDetailHeaderProps) => {
  const wsPath = useWorkspacePath();
  return (
    <header className="space-y-3">
      <Link to={wsPath("/access/roles")} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground lg:hidden">
        <ChevronLeft aria-hidden className="size-4" />
        Roles
      </Link>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <h2 className="text-base font-semibold text-balance break-words">
            {owner && <CrownIcon aria-hidden className="mr-1.5 inline size-4 -translate-y-px text-muted-foreground" />}
            {title}
          </h2>
          {line && <p className="max-w-prose text-sm text-pretty text-muted-foreground">{line}</p>}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
    </header>
  );
};
