import { ScrollView, View } from "react-native";

import { splitMessageBody, type MessageBodySegment } from "@nexul/client-core/chat";

import { MessageImage } from "@/components/chat/MessageImage";
import { MessageMarkdown } from "@/components/chat/MessageMarkdown";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";

// A fenced block keeps its lines as typed and scrolls sideways rather than wrapping; on your bubble it takes the brand's ink.
const MessageCode = ({ code, own }: { code: string; own: boolean }) => (
  <View className={cn("rounded-md border px-2.5 py-1.5", own ? "border-brand-foreground/20 bg-brand-foreground/10" : "border-border bg-surface-2")}>
    <ScrollView horizontal showsHorizontalScrollIndicator={false}>
      <Text className={cn("font-mono text-[13px] leading-5", own && "text-brand-foreground")}>{code}</Text>
    </ScrollView>
  </View>
);

interface MessageSegmentProps {
  segment: MessageBodySegment;
  own: boolean;
}

const MessageSegment = ({ segment, own }: MessageSegmentProps) => (
  <>
    {segment.kind === "image" && <MessageImage src={segment.src} alt={segment.alt} />}
    {segment.kind === "code" && <MessageCode code={segment.code} own={own} />}
    {segment.kind === "text" && <MessageMarkdown markdown={segment.text} own={own} />}
  </>
);

interface MessageBodyProps {
  body: string;
  own?: boolean;
}

export const MessageBody = ({ body, own = false }: MessageBodyProps) =>
  splitMessageBody(body).map((segment, i) => <MessageSegment key={i} segment={segment} own={own} />);
