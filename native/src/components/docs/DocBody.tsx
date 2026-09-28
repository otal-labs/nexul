import { useRouter } from "expo-router";
import { Linking } from "react-native";

import { MessageBody } from "@/components/chat/MessageBody";
import { richBodyToMarkdown } from "@/models/Doc";

interface DocBodyProps {
  body: string;
}

// Doc bodies are canonical Tiptap JSON (ADR 0026); richBodyToMarkdown converts them so the doc reader renders
// through the same markdown component Chat and the ticket screen use, rather than a second hand-rolled one.
export const DocBody = ({ body }: DocBodyProps) => {
  const router = useRouter();

  // A link to another doc or ticket pushes onto this app's own stack instead of leaving it.
  const onLinkPress = (url: string) => {
    const docMatch = /^\/docs\/([^/]+)$/.exec(url);
    if (docMatch?.[1]) {
      router.push(`/more/docs/${docMatch[1]}`);
      return;
    }
    const ticketMatch = /^\/tickets\/([^/]+)$/.exec(url);
    if (ticketMatch?.[1]) {
      router.push(`/board/ticket/${ticketMatch[1]}`);
      return;
    }
    void Linking.openURL(url);
  };

  return <MessageBody body={richBodyToMarkdown(body)} onLinkPress={onLinkPress} />;
};
