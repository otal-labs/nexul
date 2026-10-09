import type { Reaction } from "@nexul/client-core/chat";
import { personLabel } from "@nexul/client-core/person";

import { useFetchMe } from "@/hooks/AuthHooks";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { useToggleReaction } from "@/hooks/ReactionHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { Message } from "@/models/Chat";
import { cn } from "@/lib/utils";

interface ReactionChipProps {
  reaction: Reaction;
  mine: boolean;
  names: string;
  onToggle: () => void;
}

const ReactionChip = ({ reaction, mine, names, onToggle }: ReactionChipProps) => (
  <button
    type="button"
    aria-pressed={mine}
    aria-label={`${reaction.emoji} by ${names}`}
    title={names}
    onClick={onToggle}
    className={cn(
      "inline-flex h-6 items-center gap-1 rounded-full border px-2 text-xs transition-colors duration-150 ease-standard",
      mine ? "border-ring/50 bg-accent" : "border-border bg-card hover:bg-accent/40",
    )}
  >
    <span aria-hidden>{reaction.emoji}</span>
    <span className="font-mono text-muted-foreground tabular-nums">{reaction.user_ids.length}</span>
  </button>
);

const ReactionChipRow = ({ message, reactions }: { message: Message; reactions: Reaction[] }) => {
  const { data: me } = useFetchMe();
  const resolvePerson = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const toggle = useToggleReaction(message);
  return (
    <div data-slot="reactions" className="-mt-1 flex flex-wrap gap-1">
      {reactions.map((r) => (
        <ReactionChip
          key={r.emoji}
          reaction={r}
          mine={!!me && r.user_ids.includes(me.user.id)}
          names={r.user_ids.map((id) => personLabel(resolvePerson(id))).join(", ")}
          onToggle={() => toggle(r.emoji)}
        />
      ))}
    </div>
  );
};

// The reactions under a message, one chip per emoji that toggles your own; a message nobody reacted to fetches nothing.
export const MessageReactions = ({ message }: { message: Message }) => {
  const reactions = message.reactions ?? [];
  return reactions.length > 0 && <ReactionChipRow message={message} reactions={reactions} />;
};
