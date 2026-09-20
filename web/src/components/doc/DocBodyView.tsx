import { useEffect, useMemo } from "react";
import { EditorContent, useEditor } from "@tiptap/react";

import {
  buildEditorExtensions,
  extractMentionRefs,
} from "@/components/doc/mention/editorExtensions";
import { publishChips } from "@/components/doc/mention/mentionChipsStore";
import { useResolveMentions } from "@/hooks/MentionHooks";
import type { MentionChipData } from "@/models/Mention";
import { isStructuredBody, parseBodyToJSON } from "@/utils/RichtextUtility";

interface DocBodyViewProps {
  body: string;
}

// Resolves every mention ref in one batched request before publishing chips.
export const DocBodyView = ({ body }: DocBodyViewProps) => {
  const editor = useEditor({
    extensions: buildEditorExtensions(),
    content: isStructuredBody(body) ? JSON.parse(body) : body,
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
    <div className="doc-body-view doc-readonly prose-rich">
      <EditorContent editor={editor} />
    </div>
  );
};
