import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { usePersonLookup } from "@/hooks/PeopleHooks";

// A member as their avatar and name, the same display a ticket's developer gets.
export const MeetingPerson = ({ userId, workspaceId }: { userId: string; workspaceId: string }) => {
  const person = usePersonLookup(workspaceId)(userId);
  return (
    <>
      <PersonAvatar person={person} size={22} />
      <Text numberOfLines={1} className="shrink text-sm">
        {personLabel(person)}
      </Text>
    </>
  );
};
