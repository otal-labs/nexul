import { useSyncExternalStore } from "react";
import { NodeViewWrapper, type NodeViewProps } from "@tiptap/react";

import { MentionChip } from "@/components/doc/mention/MentionChip";
import { PersonMentionChip } from "@/components/doc/mention/PersonMentionChip";
import { getChips, subscribeChips } from "@/components/doc/mention/mentionChipsStore";
import type { MentionSearchType } from "@/models/Mention";

// Falls back to the stored label until the batched resolve publishes live chip data.
export const MentionNodeView = ({ node }: NodeViewProps) => {
  const chips = useSyncExternalStore(subscribeChips, getChips);
  const type = node.attrs.type as MentionSearchType | undefined;
  const id = node.attrs.id as string | undefined;
  if (!type || !id) {
    return null;
  }
  const label = (node.attrs.label as string) ?? id;
  return (
    <NodeViewWrapper as="span">
      {type === "person" && <PersonMentionChip id={id} label={label} />}
      {type !== "person" && <MentionChip type={type} id={id} label={label} chip={chips.get(`${type}:${id}`)} />}
    </NodeViewWrapper>
  );
};
