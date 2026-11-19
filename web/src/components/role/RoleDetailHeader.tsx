import type { ReactNode, Ref } from "react";
import { ChevronLeft, CrownIcon } from "lucide-react";
import { Link } from "react-router";

import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { cn } from "@/lib/utils";

interface RoleDetailHeaderProps {
  ref?: Ref<HTMLElement>;
  title: string;
  line: string;
  owner?: boolean;
  actions?: ReactNode;
  // What the role holds at a glance, under its line.
  tally?: ReactNode;
  // Shows the link back to the list at every width, for a pane that takes the list's place too (Compare roles).
  backAlways?: boolean;
}

// The open pane's title and actions on one line, then its line and reach, stuck to the pane's top where there is room (the pin variant) so its links stay in reach.
// Below 1024px a link back to the list leads the title's line; a pane that also takes the list's place keeps it at every width.
const backClass = "-ml-1 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground lg:hidden";

export const RoleDetailHeader = ({ ref, title, line, owner = false, actions, tally, backAlways = false }: RoleDetailHeaderProps) => {
  const wsPath = useWorkspacePath();
  return (
    <header ref={ref} className="sticky-crumbs -mx-4 unpin:relative! space-y-1.5 px-4 sm:-mr-3.5 sm:-ml-6 sm:pr-3.5 sm:pl-6">
      <div className="flex min-h-8 items-center justify-between gap-3">
        <Link to={wsPath("/access/roles")} aria-label="Back to roles" className={cn(backClass, "shrink-0", backAlways && "lg:inline-flex")}>
          <ChevronLeft aria-hidden className="size-4" />
          Roles
        </Link>
        <h2 className="min-w-0 flex-1 text-base font-semibold text-balance break-words">
          {owner && <CrownIcon aria-hidden className="mr-1.5 inline size-4 -translate-y-px text-muted-foreground" />}
          {title}
        </h2>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      {(line || tally) && (
        <div className="@container space-y-3">
          {line && <p className="max-w-2xl text-sm text-balance text-muted-foreground">{line}</p>}
          {tally}
        </div>
      )}
    </header>
  );
};
