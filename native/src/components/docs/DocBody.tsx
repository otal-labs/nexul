import { useMemo } from "react";
import { Image } from "react-native";
import { useUniwind } from "uniwind";

import { mentionChipText } from "@nexul/client-core/mention";

import { MessageBody } from "@/components/chat/MessageBody";
import { useResolveMentions } from "@/hooks/MentionHooks";
import { useMentionName } from "@/hooks/PeopleHooks";
import { useSelectedWorkspace } from "@/hooks/WorkspaceHooks";
import { linkMention, mentionRefs, richBodyToMarkdown, type MentionWriter } from "@/models/Doc";

// The markdown renderer takes only images inline, so the lock is a bundled glyph per theme (web/scripts/generateIcons.ts).
const lockImage = {
  dark: Image.resolveAssetSource(require("../../../assets/lock-dark.png")).uri,
  light: Image.resolveAssetSource(require("../../../assets/lock-light.png")).uri,
};

interface DocBodyProps {
  body: string;
}

// Doc and ticket bodies are canonical Tiptap JSON (ADR 0026); richBodyToMarkdown converts them so the reader renders
// through the same markdown component Chat uses, rather than a second hand-rolled one, with people named live.
export const DocBody = ({ body }: DocBodyProps) => {
  const workspace = useSelectedWorkspace();
  const nameFor = useMentionName(workspace?.id);
  const refs = useMemo(() => mentionRefs(body), [body]);
  const { data: chips } = useResolveMentions(workspace?.id, refs);
  const lock = useUniwind().theme === "dark" ? lockImage.dark : lockImage.light;

  // A mention the viewer can't open keeps its text, locked and inert, as the web's pill does (ADR 0148).
  const write: MentionWriter = (type, id, label) => {
    const chip = chips?.find((c) => c.type === type && c.id === id);
    const text = mentionChipText(type, workspace?.mention_chip_template ?? "", chip, label);
    if (chip?.can_open) return linkMention(type, id, text);
    return `${text} ![You don't have access](${lock})`;
  };

  return <MessageBody body={richBodyToMarkdown(body, nameFor, write)} />;
};
