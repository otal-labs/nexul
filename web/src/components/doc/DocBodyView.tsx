import { useEffect, useMemo } from "react";
import { EditorContent, useEditor } from "@tiptap/react";

import type { MentionChipData } from "@nexul/client-core/mention";

import { DiscordTimeLayer } from "@/components/doc/discordTime/DiscordTimeLayer";
import {
  buildEditorExtensions,
  extractMentionRefs,
} from "@/components/doc/mention/editorExtensions";
import { publishChips } from "@/components/doc/mention/mentionChipsStore";
import { useResolveMentions } from "@/hooks/MentionHooks";
import { cn } from "@/lib/utils";
import { isStructuredBody, parseBodyToJSON } from "@/utils/RichtextUtility";

interface DocBodyViewProps {
  body: string;
  /** `text-sm` or `text-xs` sets it at a small prose size, for a body quoted in a bubble or a rail. */
  className?: string;
}

// Resolves every mention ref in one batched request before publishing chips.
export const DocBodyView = ({ body, className }: DocBodyViewProps) => {
  const editor = useEditor({
    extensions: buildEditorExtensions(),
    content: isStructuredBody(body) ? parseBodyToJSON(body) : body,
    contentType: isStructuredBody(body) ? "json" : "markdown",
    editable: false,
    immediatelyRender: false,
  });

  const refs = useMemo(() => extractMentionRefs(parseBodyToJSON(body)), [body]);
  const { data } = useResolveMentions(refs);

  useEffect(() => {
    const map = new Map<string, MentionChipData>();
    for (const chip of data ?? []) map.set(`${chip.type}:${chip.id}`, chip);
    publishChips(map);
    return () => publishChips(new Map());
  }, [data]);

  return (
    <div className={cn("doc-body-view doc-readonly prose-rich", className)}>
      <DiscordTimeLayer>
        <EditorContent editor={editor} />
      </DiscordTimeLayer>
    </div>
  );
};
