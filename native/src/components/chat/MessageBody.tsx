import { MessageImage } from "@/components/chat/MessageImage";
import { MessageMarkdown } from "@/components/chat/MessageMarkdown";
import { splitMessageBody, type MessageBodySegment } from "@/models/Chat";

interface MessageSegmentProps {
  segment: MessageBodySegment;
}

const MessageSegment = ({ segment }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "text" && <MessageMarkdown markdown={segment.text} />}
  </>
);

interface MessageBodyProps {
  body: string;
}

export const MessageBody = ({ body }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} />);
