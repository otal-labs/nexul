import { Users } from "lucide-react";

import type { Conversation } from "@nexul/client-core/chat";

import { RowActions } from "@/components/listpane/RowActions";
import { PersonAvatar } from "@/components/PersonAvatar";
import { ChatSidebarRow } from "@/components/sidebar/ChatSidebarRow";
import { useFetchMe } from "@/hooks/AuthHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useBotsDialog } from "@/hooks/useBotsDialog";

interface DMSidebarRowProps {
  conversation: Conversation;
  label: string;
  unreadCount: number;
}

// A conversation with one other person leads with their avatar; a group keeps the people icon.
export const DMSidebarRow = ({ conversation, label, unreadCount }: DMSidebarRowProps) => {
  const { onBots, botsDialog } = useBotsDialog(conversation, label);
  const { data: me } = useFetchMe();
  const lookup = usePersonLookup(conversation.workspace_id);
  const others = (conversation.participant_ids ?? []).filter((id) => id !== me?.user.id);
  const person = others.length === 1 && others[0] ? lookup(others[0]) : undefined;
  return (
    <>
      <ChatSidebarRow
        conversationId={conversation.id}
        label={label}
        icon={Users}
        leading={person && <PersonAvatar login={person.login} src={person.avatar_url} label={label} className="size-[18px]" />}
        unreadCount={unreadCount}
        actions={onBots && <RowActions itemLabel={label} onBots={onBots} />}
      />
      {botsDialog}
    </>
  );
};
