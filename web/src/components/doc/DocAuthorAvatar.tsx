import { PersonAvatar } from "@/components/PersonAvatar";
import { usePerson } from "@/hooks/PeopleHooks";
import { personLabel } from "@/models/Person";

interface DocAuthorAvatarProps {
  userId: string;
}

export const DocAuthorAvatar = ({ userId }: DocAuthorAvatarProps) => {
  const person = usePerson(userId);
  return <PersonAvatar login={person.login} src={person.avatar_url} label={personLabel(person)} className="size-4 text-[8px]" />;
};
