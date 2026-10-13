import { useNavigate } from "react-router";

import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useCreateDM, useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { findDM } from "@/models/Chat";

interface OwnerRowProps {
  person: Person;
  workspaceId: string;
}

// Opens the DM with this Owner, starting one only when none exists yet.
export const OwnerRow = ({ person, workspaceId }: OwnerRowProps) => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { data: me } = useFetchMe();
  const { data: conversations } = useFetchConversations(workspaceId);
  const createDM = useCreateDM(workspaceId);
  const open = (conversationId: string) => void navigate(wsPath(`/chat/${conversationId}`));

  const message = () => {
    const existing = me && findDM(conversations ?? [], [me.user.id, person.user_id]);
    if (existing) {
      open(existing.id);
      return;
    }
    createDM.mutate([person.user_id], { onSuccess: (conversation) => open(conversation.id) });
  };

  return (
    <li>
      <button
        type="button"
        onClick={message}
        disabled={createDM.isPending}
        aria-label={`Message ${personLabel(person)}`}
        className="flex w-full items-center gap-3 px-1 py-2 text-left transition-colors duration-150 ease-standard hover:bg-accent/40"
      >
        <PersonAvatar login={person.login} src={person.avatar_url} className="size-6" />
        <span className="min-w-0 flex-1 truncate text-sm">{personLabel(person)}</span>
        <span className="font-mono text-xs text-muted-foreground">@{person.login}</span>
      </button>
    </li>
  );
};
