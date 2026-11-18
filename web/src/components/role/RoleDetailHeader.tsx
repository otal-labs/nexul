import type { ReactNode } from "react";
import { ChevronLeft, CrownIcon } from "lucide-react";
import { Link } from "react-router";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

interface RoleDetailHeaderProps {
  title: string;
  line: string;
  owner?: boolean;
  actions?: ReactNode;
  // What the role holds at a glance, under its line.
  tally?: ReactNode;
  // Shows the link back to the list at every width, for a pane that takes the list's place too (Compare roles).
  backAlways?: boolean;
}

// The open pane's title with its actions on the same line at every width, then one muted line; below 1024px, where the list
// and the pane take turns, a link back to the list.
export const RoleDetailHeader = ({ title, line, owner = false, actions, tally, backAlways = false }: RoleDetailHeaderProps) => {
  const wsPath = useWorkspacePath();
  return (
    <header className="space-y-1.5">
      <Link to={wsPath("/access/roles")} className={cn("-ml-1 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground", !backAlways && "lg:hidden")}>
        <ChevronLeft aria-hidden className="size-4" />
        Roles
      </Link>
      <div className="flex min-h-8 items-center justify-between gap-3">
        <h2 className="min-w-0 text-base font-semibold text-balance break-words">
          {owner && <CrownIcon aria-hidden className="mr-1.5 inline size-4 -translate-y-px text-muted-foreground" />}
          {title}
        </h2>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      {(line || tally) && (
        <div className="space-y-3">
          {line && <p className="max-w-prose text-sm text-pretty text-muted-foreground">{line}</p>}
          {tally}
        </div>
      )}
    </header>
  );
};
