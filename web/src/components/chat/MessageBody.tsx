import type { ClipboardEvent } from "react";

import { splitMessageBody, type MessageBodySegment } from "@nexul/client-core/chat";
import { splitDiscordTimes, type TimedTextPart } from "@nexul/client-core/discordTime";

import { DiscordTimeChip } from "@/components/chat/DiscordTimeChip";
import { MessageImage } from "@/components/chat/MessageImage";
import { MessageLink } from "@/components/chat/MessageLink";
import { tokenizeMessageText, type MessageTextPart } from "@/utils/MessageTextUtility";

// Plain text with line breaks kept, @mentions bolded, http(s) URLs linked and <t:…> codes as times; only a bot's is markdown.
const MessageTextPartView = ({ part }: { part: MessageTextPart }) => (
  <>
    {part.kind === "text" && part.text}
    {part.kind === "mention" && <strong className="font-semibold">{part.text}</strong>}
    {part.kind === "link" && <MessageLink url={part.url} />}
  </>
);

interface TimedTextProps {
  part: TimedTextPart;
  mentionHandles: string[];
}

const TimedText = ({ part, mentionHandles }: TimedTextProps) => (
  <>
    {part.kind === "time" && <DiscordTimeChip time={part.time} />}
    {part.kind === "text" &&
      tokenizeMessageText(part.text, mentionHandles).map((textPart, i) => <MessageTextPartView key={i} part={textPart} />)}
  </>
);

// A pill shows a label, so a selection inside one message copies each pill as its URL instead.
const copyPillUrls = (event: ClipboardEvent<HTMLParagraphElement>) => {
  const selection = window.getSelection();
  if (!selection || selection.rangeCount === 0) return;
  const range = selection.getRangeAt(0);
  if (!event.currentTarget.contains(range.commonAncestorContainer)) return;
  const fragment = range.cloneContents();
  const pills = fragment.querySelectorAll<HTMLElement>("[data-url]");
  if (pills.length === 0) return;
  pills.forEach((pill) => pill.replaceWith(pill.dataset.url ?? ""));
  event.clipboardData.setData("text/plain", fragment.textContent);
  event.preventDefault();
};

interface MessageSegmentProps {
  segment: MessageBodySegment;
  mentionHandles: string[];
}

const MessageSegment = ({ segment, mentionHandles }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "code" && (
      <pre className="overflow-x-auto rounded-md border border-border bg-surface-2 px-2.5 py-1.5 font-mono text-xs leading-relaxed">
        <code>{segment.code}</code>
      </pre>
    )}
    {segment.kind === "text" && (
      <p dir="auto" className="whitespace-pre-wrap" onCopy={copyPillUrls}>
        {splitDiscordTimes(segment.text).map((part, i) => (
          <TimedText key={i} part={part} mentionHandles={mentionHandles} />
        ))}
      </p>
    )}
  </>
);

interface MessageBodyProps {
  body: string;
  mentionHandles: string[];
}

export const MessageBody = ({ body, mentionHandles }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} mentionHandles={mentionHandles} />);
