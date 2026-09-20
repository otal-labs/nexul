import { useEffect, useRef } from "react";

import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
  useMessageScroller,
  useMessageScrollerVisibility,
} from "@/components/ui/message-scroller";

import { AgentStreamBubble } from "@/components/chat/AgentStreamBubble";
import { MessageRow } from "@/components/chat/MessageRow";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useThreadTrailBlocks } from "@/hooks/TrailHooks";
import type { Conversation, Message } from "@/models/Chat";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { trailBlockFor } from "@/utils/ThreadTrailUtility";

interface MessageListProps {
  conversation: Conversation;
  messages: Message[];
  currentUserId: string | undefined;
  resolveAuthorLogin: (authorId: string) => string;
  onEdit: (messageId: string, body: string) => Promise<void>;
  onDelete: (messageId: string) => Promise<void>;
  onInterruptAgent: () => void;
  // Fired when the newest persisted message enters the viewport; a scrolled-up chat is not "read".
  onNewestSeen?: ((messageId: string) => void) | undefined;
}

// IntersectionObserver only runs while this is mounted; reports the newest visible message.
const NewestSeenReporter = ({ newestId, onSeen }: { newestId: string; onSeen: (id: string) => void }) => {
  const { visibleMessageIds } = useMessageScrollerVisibility();
  const seen = visibleMessageIds.includes(newestId);
  useEffect(() => {
    if (seen) onSeen(newestId);
  }, [seen, newestId, onSeen]);
  return null;
};

// The scroller only follows the edge from the bottom; sending your own message must land the view on it regardless.
const OwnMessageScroller = ({ newestId, isOwn }: { newestId: string; isOwn: boolean }) => {
  const { scrollToEnd } = useMessageScroller();
  const previous = useRef<string | null>(null);
  useEffect(() => {
    const first = previous.current === null;
    if (previous.current === newestId) return;
    previous.current = newestId;
    if (!first && isOwn) scrollToEnd();
  }, [newestId, isOwn, scrollToEnd]);
  return null;
};

// A question is answered once the thread moved past it: the user's "Answered" reply, or the Agent speaking again.
const answeredAfter = (messages: Message[], index: number): boolean =>
  messages.slice(index + 1).some((m) => m.author_kind === "agent" || (m.author_kind === "user" && m.body.startsWith("Answered")));

// MessageScroller follows the live edge while streamed text grows, and releases the moment the reader scrolls away.
export const MessageList = ({
  conversation,
  messages,
  currentUserId,
  resolveAuthorLogin,
  onEdit,
  onDelete,
  onInterruptAgent,
  onNewestSeen,
}: MessageListProps) => {
  const stream = useAgentStreamStore((s) => s.streams[conversation.id]);
  const blocks = useThreadTrailBlocks(conversation);

  const empty = messages.length === 0 && !stream;

  return (
    <>
      {empty && (
        <div className="min-h-0 flex-1 py-2">
          <NoDataDisplay message="No messages yet — say hello" size="compact" />
        </div>
      )}
      {!empty && (
        <MessageScrollerProvider autoScroll defaultScrollPosition="last-anchor">
          <MessageScroller className="min-h-0 flex-1">
            <MessageScrollerViewport>
              <MessageScrollerContent className="py-2" aria-busy={stream?.streaming ?? false}>
                {messages.map((message, i) => (
                  // Only @Agent turns anchor to the viewport top; anchoring every message shoved chatter to the top.
                  <MessageScrollerItem
                    key={message.id}
                    messageId={message.id}
                    scrollAnchor={message.author_kind === "user" && (message.mentions ?? []).some((m) => m.kind === "agent")}
                  >
                    <MessageRow
                      message={message}
                      authorLogin={resolveAuthorLogin(message.author_id)}
                      isOwn={message.author_id === currentUserId}
                      questionAnswered={message.author_kind === "agent" && answeredAfter(messages, i)}
                      trailBlock={trailBlockFor(message, blocks)}
                      onEdit={onEdit}
                      onDelete={onDelete}
                    />
                  </MessageScrollerItem>
                ))}
                {stream && (
                  <MessageScrollerItem messageId={`stream-${conversation.id}`}>
                    <AgentStreamBubble frame={stream} onInterrupt={onInterruptAgent} live={blocks.live} />
                  </MessageScrollerItem>
                )}
              </MessageScrollerContent>
            </MessageScrollerViewport>
            <MessageScrollerButton />
            {messages.length > 0 && (
              <OwnMessageScroller
                newestId={messages[messages.length - 1]!.id}
                isOwn={messages[messages.length - 1]!.author_kind === "user" && messages[messages.length - 1]!.author_id === currentUserId}
              />
            )}
            {onNewestSeen && messages.length > 0 && (
              <NewestSeenReporter newestId={messages[messages.length - 1]!.id} onSeen={onNewestSeen} />
            )}
          </MessageScroller>
        </MessageScrollerProvider>
      )}
    </>
  );
};
