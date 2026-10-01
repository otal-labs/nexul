import { UsersIcon } from "lucide-react";

import type { RestrictedMember } from "@/models/Project";

interface RestrictedMembersLoseAccessProps {
  members: RestrictedMember[];
}

const loseLine = (names: string[]): string => {
  const others = names.length - 1;
  if (others === 0) return `${names[0]} loses access.`;
  return `${names[0]} and ${others} other restricted member${others === 1 ? "" : "s"} lose access.`;
};

// A signal in the delete confirmation, not a gate: who among the Restricted members this leaves without the project.
export const RestrictedMembersLoseAccess = ({ members }: RestrictedMembersLoseAccessProps) => {
  const names = members.map((member) => member.name);
  return (
    <div className="flex gap-3 border-y border-border py-3">
      <UsersIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0 space-y-0.5">
        <p className="text-sm font-medium">{loseLine(names)}</p>
        {names.length > 1 && <p className="text-sm break-words text-muted-foreground">{names.join(", ")}</p>}
      </div>
    </div>
  );
};
