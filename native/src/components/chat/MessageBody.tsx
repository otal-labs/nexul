import { MessageImage } from "@/components/chat/MessageImage";
import { MessageMarkdown } from "@/components/chat/MessageMarkdown";
import { splitMessageBody, type MessageBodySegment } from "@/models/Chat";

interface MessageSegmentProps {
  segment: MessageBodySegment;
  onLinkPress?: (url: string) => void;
}

const MessageSegment = ({ segment, onLinkPress }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "text" && <MessageMarkdown markdown={segment.text} {...(onLinkPress && { onLinkPress })} />}
  </>
);

interface MessageBodyProps {
  body: string;
  /** Overrides the default "open in the system browser"; a caller with internal links routes them here. */
  onLinkPress?: (url: string) => void;
}

export const MessageBody = ({ body, onLinkPress }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} {...(onLinkPress && { onLinkPress })} />);
