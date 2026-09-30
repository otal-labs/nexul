import { MessageBody } from "@/components/chat/MessageBody";
import { useMentionName } from "@/hooks/PeopleHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { richBodyToMarkdown } from "@/models/Doc";

interface DocBodyProps {
  body: string;
}

// Doc and ticket bodies are canonical Tiptap JSON (ADR 0026); richBodyToMarkdown converts them so the reader renders
// through the same markdown component Chat uses, rather than a second hand-rolled one, with people named live.
export const DocBody = ({ body }: DocBodyProps) => {
  const nameFor = useMentionName(useCurrentWorkspaceId());
  return <MessageBody body={richBodyToMarkdown(body, nameFor)} />;
};
