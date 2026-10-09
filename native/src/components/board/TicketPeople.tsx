import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";

// A ticket role's holder as their avatar and name, or a muted "No one".
export const TicketPerson = ({ login }: { login: string }) => {
  const resolvePerson = usePersonLookup(useCurrentWorkspaceId());
  const person = resolvePerson(login);
  return (
    <>
      {login === "" && <Text className="text-sm text-muted-foreground">No one</Text>}
      {login !== "" && <PersonAvatar person={person} size={22} />}
      {login !== "" && (
        <Text numberOfLines={1} className="shrink text-sm">
          {personLabel(person)}
        </Text>
      )}
    </>
  );
};
