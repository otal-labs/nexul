import { MessageBody } from "@/components/chat/MessageBody";
import { richBodyToMarkdown } from "@/models/Doc";

interface DocBodyProps {
  body: string;
}

// Doc bodies are canonical Tiptap JSON (ADR 0026); richBodyToMarkdown converts them so the doc reader renders
// through the same markdown component Chat and the ticket screen use, rather than a second hand-rolled one.
export const DocBody = ({ body }: DocBodyProps) => <MessageBody body={richBodyToMarkdown(body)} />;
