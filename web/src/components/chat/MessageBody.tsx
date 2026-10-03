import type { ClipboardEvent } from "react";

import { MessageImage } from "@/components/chat/MessageImage";
import { MessageLink } from "@/components/chat/MessageLink";
import { splitMessageBody, type MessageBodySegment } from "@/models/Chat";
import { tokenizeMessageText, type MessageTextPart } from "@/utils/MessageTextUtility";

// Plain text with line breaks kept, each @mention bolded, and http(s) URLs linked; no markdown-string renderer exists in the app yet.
const MessageTextPartView = ({ part }: { part: MessageTextPart }) => (
  <>
    {part.kind === "text" && part.text}
    {part.kind === "mention" && <strong className="font-semibold">{part.text}</strong>}
    {part.kind === "link" && <MessageLink url={part.url} />}
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
      <p className="whitespace-pre-wrap" onCopy={copyPillUrls}>
        {tokenizeMessageText(segment.text, mentionHandles).map((part, i) => (
          <MessageTextPartView key={i} part={part} />
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
