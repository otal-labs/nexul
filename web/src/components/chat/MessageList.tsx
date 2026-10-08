import { useEffect, useRef, useState } from "react";

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
import { ChatDayDivider } from "@/components/chat/ChatDayDivider";
import { ChatPaneState } from "@/components/chat/ChatPaneState";
import { MessageRow } from "@/components/chat/MessageRow";
import { playSend } from "@/components/chat/sendMotion";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useThreadTrailBlocks } from "@/hooks/TrailHooks";
import { cn } from "@/lib/utils";
import { isContinuation, isNote, type Conversation, type Message } from "@/models/Chat";
import type { Person } from "@/models/Person";
import { useAgentStreamStore } from "@/stores/agentStreamStore";
import { startsDay } from "@/utils/ChatDayUtility";
import { trailBlockFor } from "@/utils/ThreadTrailUtility";

interface MessageListProps {
  conversation: Conversation;
  messages: Message[];
  currentUserId: string | undefined;
  resolveAuthor: (authorId: string) => Person;
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
  messages
    .slice(index + 1)
    .some((m) => (m.author_kind === "agent" && !isNote(m)) || (m.author_kind === "user" && m.body.startsWith("Answered")));

const rowKey = (message: Message) => message.client_key ?? message.id;

type Entrance = "send" | "arrive";

// After the conversation opens: yours plays the send, others' rise in, the Agent's arrives as its stream bubble instead.
const entranceOf = (message: Message, seen: Set<string>, own: boolean): Entrance | undefined => {
  if (seen.has(rowKey(message))) return undefined;
  if (own) return "send";
  if (message.author_kind === "agent") return undefined;
  return "arrive";
};

// MessageScroller follows the live edge while streamed text grows, and releases the moment the reader scrolls away.
export const MessageList = ({
  conversation,
  messages,
  currentUserId,
  resolveAuthor,
  onEdit,
  onDelete,
  onInterruptAgent,
  onNewestSeen,
}: MessageListProps) => {
  const stream = useAgentStreamStore((s) => s.streams[conversation.id]);
  const blocks = useThreadTrailBlocks(conversation);

  const [seen] = useState(() => new Set(messages.map(rowKey)));
  const [streamAtOpen] = useState(() => stream !== undefined);
  const empty = messages.length === 0 && !stream;

  return (
    <>
      {empty && (
        <ChatPaneState>
          <NoDataDisplay message="No messages yet — say hello" size="compact" />
        </ChatPaneState>
      )}
      {!empty && (
        <MessageScrollerProvider autoScroll defaultScrollPosition="last-anchor">
          <MessageScroller className="min-h-0 flex-1">
            <MessageScrollerViewport>
              <MessageScrollerContent className="mx-auto w-full max-w-3xl gap-0 py-2" aria-busy={stream?.streaming ?? false}>
                {messages.map((message, i) => {
                  const continuation = isContinuation(messages[i - 1], message);
                  const own = message.author_id === currentUserId;
                  const entrance = entranceOf(message, seen, own);
                  const newDay = startsDay(messages[i - 1], message);
                  return (
                    // Only the newest message, when it is an @Agent turn, anchors: the scroller jumps to any older anchor on a same-count swap (pending row confirmed, stream bubble replaced).
                    <MessageScrollerItem
                      key={rowKey(message)}
                      ref={entrance === "send" ? playSend : undefined}
                      messageId={message.id}
                      // The item's content-visibility clips paint to its box; the margin lets the Edit/Delete pill rise into the gap above.
                      // The newest row renders eagerly: its 10rem placeholder would park a just-sent message above the bottom edge.
                      className={cn(
                        continuation ? "pt-1.5 [overflow-clip-margin:1rem]" : "pt-4",
                        i === messages.length - 1 && "[content-visibility:visible]",
                        entrance === "arrive" && "arrive",
                      )}
                      scrollAnchor={
                        i === messages.length - 1 && message.author_kind === "user" && (message.mentions ?? []).some((m) => m.kind === "agent")
                      }
                    >
                      {newDay && <ChatDayDivider createdAt={message.created_at} />}
                      <MessageRow
                        message={message}
                        author={resolveAuthor(message.author_id)}
                        isOwn={own}
                        continuation={continuation}
                        questionAnswered={message.author_kind === "agent" && answeredAfter(messages, i)}
                        trailBlock={trailBlockFor(message, blocks)}
                        ticketId={conversation.ticket_id}
                        onEdit={onEdit}
                        onDelete={onDelete}
                      />
                    </MessageScrollerItem>
                  );
                })}
                {stream && (
                  <MessageScrollerItem messageId={`stream-${conversation.id}`} className={cn("pt-5 [content-visibility:visible]", !streamAtOpen && "arrive")}>
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
