import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePerson } from "@/hooks/PeopleHooks";

interface DocWatcherRowProps {
  userId: string;
  isYou: boolean;
}

export const DocWatcherRow = ({ userId, isYou }: DocWatcherRowProps) => {
  const person = usePerson(userId);
  const label = personLabel(person);

  return (
    <li className="flex items-center gap-2 px-3 py-1.5 text-sm">
      <PersonAvatar login={person.login} src={person.avatar_url} label={label} />
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {isYou && <span className="font-mono text-xs text-muted-foreground">you</span>}
    </li>
  );
};
