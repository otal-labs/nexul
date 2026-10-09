import { MessageImage } from "@/components/chat/MessageImage";
import { MessageMarkdown } from "@/components/chat/MessageMarkdown";
import { splitMessageBody, type MessageBodySegment } from "@/models/Chat";

interface MessageSegmentProps {
  segment: MessageBodySegment;
  own: boolean;
}

const MessageSegment = ({ segment, own }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "text" && <MessageMarkdown markdown={segment.text} own={own} />}
  </>
);

interface MessageBodyProps {
  body: string;
  own?: boolean;
}

export const MessageBody = ({ body, own = false }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} own={own} />);
