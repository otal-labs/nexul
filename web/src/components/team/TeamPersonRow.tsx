import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import type { TeamPerson } from "@/models/Team";
import { accessSummary, personName } from "@/utils/TeamUtility";

interface TeamPersonRowProps {
  person: TeamPerson;
  index: number;
  onOpen: (id: string) => void;
}

export const TeamPersonRow = ({ person, index, onOpen }: TeamPersonRowProps) => {
  const summary = accessSummary(person);
  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(person.id)}
        aria-label={`Open ${personName(person)}`}
        className="animate-in fade-in-0 slide-in-from-bottom-1 flex w-full items-center gap-3 px-3 py-3 text-left duration-150 ease-out transition-colors hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:px-4"
        style={{ animationDelay: `${Math.min(index, 7) * 25}ms` }}
      >
        <PersonAvatar login={person.login} src={person.avatar_url} className="size-8" />
        <span className="min-w-0 flex-1">
          <span className="block truncate font-medium">{personName(person)}</span>
          <span className="block truncate font-mono text-xs text-muted-foreground">@{person.login}</span>
        </span>
        <span className="hidden min-w-0 flex-1 truncate text-right text-sm text-muted-foreground md:block">
          {summary || "No workspace access"}
        </span>
        <AccountStatusLabel status={person.status} />
        <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      </button>
    </li>
  );
};
