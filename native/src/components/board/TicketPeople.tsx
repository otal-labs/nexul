import { personLabel } from "@nexul/client-core/person";
import type { TicketPersonRef } from "@nexul/client-core/ticketPeople";

import { PersonAvatar } from "@/components/PersonAvatar";
import { Text } from "@/components/ui/text";
import { useTicketPersonLookup } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";

// A ticket role's holder as their avatar and name, or a muted "No one".
export const TicketPerson = ({ holder }: { holder: TicketPersonRef }) => {
  const person = useTicketPersonLookup(useCurrentWorkspaceId())(holder);
  return (
    <>
      {holder.key === "" && <Text className="text-sm text-muted-foreground">No one</Text>}
      {holder.key !== "" && <PersonAvatar person={person} size={22} />}
      {holder.key !== "" && (
        <Text numberOfLines={1} className="shrink text-sm">
          {personLabel(person)}
        </Text>
      )}
    </>
  );
};
