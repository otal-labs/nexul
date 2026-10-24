import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePerson } from "@/hooks/PeopleHooks";

export const MeetingPersonRow = ({ userId }: { userId: string }) => {
  const person = usePerson(userId);
  return (
    <li className="flex items-center gap-2 py-1 text-sm">
      <PersonAvatar login={person.login} src={person.avatar_url} className="size-6" />
      <span className="min-w-0 flex-1 truncate">{personLabel(person)}</span>
    </li>
  );
};
