import { useSyncExternalStore } from "react";
import { NodeViewWrapper, type NodeViewProps } from "@tiptap/react";

import { MentionChip } from "@/components/doc/mention/MentionChip";
import { getChips, subscribeChips } from "@/components/doc/mention/mentionChipsStore";
import type { MentionType } from "@/models/Mention";

// Falls back to the stored label until the batched resolve publishes live chip data.
export const MentionNodeView = ({ node }: NodeViewProps) => {
  const chips = useSyncExternalStore(subscribeChips, getChips);
  const type = node.attrs.type as MentionType | undefined;
  const id = node.attrs.id as string | undefined;
  if (!type || !id) {
    return null;
  }
  const chip = chips.get(`${type}:${id}`);
  return (
    <NodeViewWrapper as="span">
      <MentionChip type={type} id={id} label={(node.attrs.label as string) ?? id} chip={chip} />
    </NodeViewWrapper>
  );
};
