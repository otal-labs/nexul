import { Message, MessageContent, MessageHeader } from "@/components/ui/message";

import { formatRelativeTime } from "@/utils/TimeUtility";

interface TrailUserBubbleProps {
  body: string;
  // at is null for an answer: the trail keeps when it was asked, not when it was answered.
  at: string | null;
}

// The starter's side of the transcript, in the chat's own-message shape: right-aligned, compact, no name.
export const TrailUserBubble = ({ body, at }: TrailUserBubbleProps) => (
  <Message align="end" className="px-1 py-1">
    <MessageContent>
      {at !== null && (
        <MessageHeader className="justify-end px-1">
          <span className="shrink-0 font-mono text-[11px]">{formatRelativeTime(at)}</span>
        </MessageHeader>
      )}
      <div
        data-slot="bubble"
        className="max-w-[85%] rounded-2xl rounded-br-md bg-accent px-3 py-2 text-sm break-words whitespace-pre-wrap text-accent-foreground"
      >
        {body}
      </div>
    </MessageContent>
  </Message>
);
