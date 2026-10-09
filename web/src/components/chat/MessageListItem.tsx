import { memo } from "react";

import { ChatDayDivider } from "@/components/chat/ChatDayDivider";
import { MessageRow } from "@/components/chat/MessageRow";
import { playSend } from "@/components/chat/sendMotion";
import { MessageScrollerItem } from "@/components/ui/message-scroller";
import { cn } from "@/lib/utils";
import type { Message } from "@/models/Chat";
import type { Person } from "@/models/Person";
import type { TrailBlock } from "@/utils/ThreadTrailUtility";

export type Entrance = "send" | "arrive";

interface MessageListItemProps {
  message: Message;
  author: Person;
  isOwn: boolean;
  continuation: boolean;
  newDay: boolean;
  newest: boolean;
  entrance: Entrance | undefined;
  scrollAnchor: boolean;
  questionAnswered: boolean;
  trailBlock: TrailBlock | undefined;
  ticketId: string | undefined;
  onEdit: (messageId: string, body: string) => Promise<void>;
  onDelete: (messageId: string) => Promise<void>;
}

// One message in the scroller, memoized so a new message or a streamed token renders only the items it changed.
export const MessageListItem = memo(
  ({ message, author, isOwn, continuation, newDay, newest, entrance, scrollAnchor, questionAnswered, trailBlock, ticketId, onEdit, onDelete }: MessageListItemProps) => (
    <MessageScrollerItem
      ref={entrance === "send" ? playSend : undefined}
      messageId={message.id}
      // The item's content-visibility clips paint to its box; the margin lets the Edit/Delete pill rise into the gap above.
      // The newest row renders eagerly: its 10rem placeholder would park a just-sent message above the bottom edge.
      className={cn(
        continuation ? "pt-1.5 [overflow-clip-margin:1rem]" : "pt-4",
        newest && "[content-visibility:visible]",
        entrance === "arrive" && "arrive",
      )}
      scrollAnchor={scrollAnchor}
    >
      {newDay && <ChatDayDivider createdAt={message.created_at} />}
      <MessageRow
        message={message}
        author={author}
        isOwn={isOwn}
        continuation={continuation}
        questionAnswered={questionAnswered}
        trailBlock={trailBlock}
        ticketId={ticketId}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </MessageScrollerItem>
  ),
);
