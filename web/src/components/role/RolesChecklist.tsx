import { useEffect, useRef, useState } from "react";
import { TriangleAlert } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { RoleCheckRow } from "@/components/role/RoleCheckRow";
import { RoleFindField } from "@/components/role/RoleFindField";
import { RoleReachHead } from "@/components/role/RoleReach";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useRoleChoices } from "@/hooks/RoleHooks";
import { cn } from "@/lib/utils";

// Past this many roles you may give, a find field leads the list, as on a role's holders.
const FIND_FROM = 8;

interface RolesChecklistProps {
  workspaceId: string;
  value: string[];
  onChange: (roleIds: string[]) => void;
  title: string;
  // Said under the list while nothing is ticked: a warning for a member, a request on an invitation.
  emptyNote: string;
}

// Every role in the workspace at once, the ones you may give first; a person holds what their roles allow, combined.
export const RolesChecklist = ({ workspaceId, value, onChange, title, emptyNote }: RolesChecklistProps) => {
  const { choices, error } = useRoleChoices(workspaceId);
  const { data: catalog } = useFetchPermissionCatalog();
  // The note settles in when the last role goes, never on a list that opens empty: one entrance at a time.
  const [startedEmpty] = useState(value.length === 0);
  const toggle = (roleId: string) => onChange(value.includes(roleId) ? value.filter((id) => id !== roleId) : [...value, roleId]);
  const [query, setQuery] = useState("");
  const needle = query.trim().toLowerCase();
  const allGivable = choices?.filter((choice) => choice.givable) ?? [];
  const givable = allGivable.filter((choice) => !needle || `${choice.role.name} ${choice.role.description}`.toLowerCase().includes(needle));
  // Opened from a role low in the list, the dialog brings that ticked role into view once, so the pick is never below the fold.
  const shown = useRef(false);
  const listRef = useRef<HTMLUListElement>(null);
  useEffect(() => {
    if (shown.current || !choices) return;
    shown.current = true;
    listRef.current?.querySelector("[data-state=checked]")?.closest("li")?.scrollIntoView?.({ block: "nearest" });
  }, [choices]);
  const locked = choices?.filter((choice) => !choice.givable) ?? [];
  // Counts only what you may give, so a role like Owner never makes the list look unfinished.
  const ticked = allGivable.filter((choice) => value.includes(choice.role.id)).length;

  return (
    <section className="space-y-1">
      <div className="flex items-baseline justify-between gap-3">
        <Microheader>{title}</Microheader>
        {choices && (
          <span className="font-mono text-xs text-muted-foreground tabular-nums">
            {ticked} of {allGivable.length}
          </span>
        )}
      </div>
      <p className="text-xs text-muted-foreground">Everything these roles allow, combined.</p>
      {allGivable.length > FIND_FROM && <RoleFindField value={query} onChange={setQuery} placeholder="Search roles" label="Find a role" className="mt-2" />}
      {needle && givable.length === 0 && <p className="py-3 text-sm text-muted-foreground">No role matches.</p>}
      {/* The column names stay at the top of the dialog's scroll, so a cell far down still says its section. */}
      {givable.length > 0 && <RoleReachHead className="sticky -top-1 z-10 bg-popover pt-2 pb-1 pl-7" />}
      {!choices && !error && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {givable.length > 0 && (
        <ul ref={listRef} className="divide-y divide-border border-b border-border">
          {givable.map((choice) => (
            <RoleCheckRow key={choice.role.id} choice={choice} catalog={catalog ?? []} checked={value.includes(choice.role.id)} onToggle={toggle} />
          ))}
        </ul>
      )}
      {locked.length > 0 && (
        <div className="pt-4">
          <Microheader>Can&apos;t give</Microheader>
          <ul className="divide-y divide-border">
            {locked.map((choice) => (
              <RoleCheckRow key={choice.role.id} choice={choice} catalog={catalog ?? []} checked={value.includes(choice.role.id)} onToggle={toggle} />
            ))}
          </ul>
        </div>
      )}
      {choices && value.length === 0 && (
        <p className={cn(!startedEmpty && "settle-in", "flex items-start gap-2 pt-3 text-xs text-muted-foreground")}>
          <TriangleAlert aria-hidden className="mt-px size-3.5 shrink-0 text-warning" />
          <span>{emptyNote}</span>
        </p>
      )}
    </section>
  );
};
