import type { Conversation } from "@nexul/client-core/chat";
import type { Person } from "@nexul/client-core/person";

import { useFetchMe } from "@/hooks/AuthHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";

/** The other person in a conversation with exactly one, whose face leads it; undefined for a group or any channel. */
export const useDMPeer = (conversation: Conversation): Person | undefined => {
  const { data: me } = useFetchMe();
  const lookup = usePersonLookup(conversation.workspace_id);
  if (conversation.kind !== "dm") return undefined;
  const others = (conversation.participant_ids ?? []).filter((id) => id !== me?.user.id);
  return others.length === 1 && others[0] ? lookup(others[0]) : undefined;
};
