import { ArrowRight } from "lucide-react";

import { personLabel } from "@nexul/client-core/person";

import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { usePerson } from "@/hooks/PeopleHooks";
import type { ImportPerson } from "@/models/Import";

interface ImportPersonRowProps {
  person: ImportPerson;
}

// One person the source names, and the member their tickets are assigned to.
export const ImportPersonRow = ({ person }: ImportPersonRowProps) => {
  const member = usePerson(person.member);
  return (
    <li className="flex min-h-11 items-center gap-3 py-2 text-sm">
      <span className="min-w-0 flex-1 truncate font-mono">{person.login ?? person.email ?? person.username ?? person.name}</span>
      <ArrowRight className="size-3.5 shrink-0 text-muted-foreground" aria-label="maps to" />
      <span className="min-w-0 flex-1 truncate">
        {person.member && personLabel(member)}
        {!person.member && <SettingsStatus tone="muted">Unmatched</SettingsStatus>}
      </span>
    </li>
  );
};
