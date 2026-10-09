import { ChevronRight } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { AccountStatusLabel } from "@/components/team/AccountStatusLabel";
import type { TeamPerson } from "@/models/Team";
import { personName, presenceText } from "@/utils/TeamUtility";

interface TeamPersonRowProps {
  person: TeamPerson;
  onOpen: (id: string) => void;
}

export const TeamPersonRow = ({ person, onOpen }: TeamPersonRowProps) => (
    <li>
      <button
        type="button"
        onClick={() => onOpen(person.id)}
        aria-label={`Open ${personName(person)}`}
        className="flex w-full items-center gap-3 px-3 py-3 text-left transition-colors duration-150 ease-standard hover:bg-accent/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:px-4"
      >
        <PersonAvatar login={person.login} src={person.avatar_url} className="size-8" />
        <span className="min-w-0 flex-1">
          <span className="block truncate font-medium">{personName(person)}</span>
          <span className="block truncate font-mono text-xs text-muted-foreground">@{person.login}</span>
        </span>
        <span className="hidden shrink-0 text-right text-sm text-muted-foreground md:block">
          {presenceText(person)}
        </span>
        <AccountStatusLabel status={person.status} online={person.online} />
        <ChevronRight aria-hidden className="size-4 shrink-0 text-muted-foreground" />
      </button>
    </li>
);
