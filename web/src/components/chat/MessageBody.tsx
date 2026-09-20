import type { ReactNode } from "react";

import { MessageImage } from "@/components/chat/MessageImage";
import { splitMessageBody, type MessageBodySegment } from "@/models/Chat";

// Plain text with line breaks kept and each @mention bolded; no markdown-string renderer exists in the app yet.
const renderTextSegment = (text: string, mentionHandles: string[]): ReactNode => {
  if (mentionHandles.length === 0) return text;
  const escaped = mentionHandles.map((h) => h.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const pattern = new RegExp(`@(?:${escaped.join("|")})\\b`, "gi");
  const parts = text.split(pattern);
  const matches = text.match(pattern) ?? [];
  return parts.map((part, i) => (
    <span key={i}>
      {part}
      {matches[i] && <strong className="font-semibold">{matches[i]}</strong>}
    </span>
  ));
};

interface MessageSegmentProps {
  segment: MessageBodySegment;
  mentionHandles: string[];
}

const MessageSegment = ({ segment, mentionHandles }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "text" && <p className="whitespace-pre-wrap">{renderTextSegment(segment.text, mentionHandles)}</p>}
  </>
);

interface MessageBodyProps {
  body: string;
  mentionHandles: string[];
}

export const MessageBody = ({ body, mentionHandles }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} mentionHandles={mentionHandles} />);
