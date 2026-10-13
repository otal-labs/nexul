import { useMemo, useState } from "react";
import { TriangleAlert } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Microheader } from "@/components/Microheader";
import { RoleCheckRow } from "@/components/role/RoleCheckRow";
import { useFetchPermissionCatalog } from "@/hooks/PermissionHooks";
import { useRoleChoices } from "@/hooks/RoleHooks";
import { cn } from "@/lib/utils";
import { domainsOf } from "@/models/PermissionLevel";

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
  const domains = useMemo(() => domainsOf(catalog ?? []), [catalog]);
  // The note settles in when the last role goes, never on a list that opens empty: one entrance at a time.
  const [startedEmpty] = useState(value.length === 0);
  const toggle = (roleId: string) => onChange(value.includes(roleId) ? value.filter((id) => id !== roleId) : [...value, roleId]);
  const givable = choices?.filter((choice) => choice.givable) ?? [];
  const locked = choices?.filter((choice) => !choice.givable) ?? [];

  return (
    <section className="space-y-1">
      <div className="flex items-baseline justify-between gap-3">
        <Microheader>{title}</Microheader>
        {choices && (
          <span className="font-mono text-xs text-muted-foreground tabular-nums">
            {value.length} of {choices.length}
          </span>
        )}
      </div>
      <p className="text-xs text-muted-foreground">Everything these roles allow, combined.</p>
      {!choices && !error && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {givable.length > 0 && (
        <ul className="divide-y divide-border border-b border-border">
          {givable.map((choice) => (
            <RoleCheckRow key={choice.role.id} choice={choice} domains={domains} checked={value.includes(choice.role.id)} onToggle={toggle} />
          ))}
        </ul>
      )}
      {locked.length > 0 && (
        <div className="pt-4">
          <Microheader>Can&apos;t give</Microheader>
          <ul className="divide-y divide-border">
            {locked.map((choice) => (
              <RoleCheckRow key={choice.role.id} choice={choice} domains={domains} checked={value.includes(choice.role.id)} onToggle={toggle} />
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
