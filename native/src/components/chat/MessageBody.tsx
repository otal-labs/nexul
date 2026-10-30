import { ScrollView, View } from "react-native";

import { splitMessageBody, type MessageBodySegment } from "@nexul/client-core/chat";

import { MessageImage } from "@/components/chat/MessageImage";
import { MessageMarkdown } from "@/components/chat/MessageMarkdown";
import { Text } from "@/components/ui/text";

// A fenced block keeps its lines as typed and scrolls sideways rather than wrapping.
const MessageCode = ({ code }: { code: string }) => (
  <View className="rounded-md border border-border bg-surface-2 px-2.5 py-1.5">
    <ScrollView horizontal showsHorizontalScrollIndicator={false}>
      <Text className="font-mono text-[13px] leading-5">{code}</Text>
    </ScrollView>
  </View>
);

interface MessageSegmentProps {
  segment: MessageBodySegment;
}

const MessageSegment = ({ segment }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "code" && <MessageCode code={segment.code} />}
    {segment.kind === "text" && <MessageMarkdown markdown={segment.text} />}
  </>
);

interface MessageBodyProps {
  body: string;
}

export const MessageBody = ({ body }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} />);
