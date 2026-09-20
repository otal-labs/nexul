import { Extension, type Editor } from "@tiptap/core";
import Collaboration from "@tiptap/extension-collaboration";
import { useQueryClient } from "@tanstack/react-query";
import { EditorContent, useEditor, useEditorState } from "@tiptap/react";
import { yCursorPlugin } from "@tiptap/y-tiptap";
import { useEffect, useMemo, useRef } from "react";
import type { Awareness } from "y-protocols/awareness";
import type * as Y from "yjs";

import { FormatBubbleMenu } from "@/components/doc/FormatBubbleMenu";
import { extractHeadingsFromJSON, type DocHeading } from "@/components/doc/docHeadings";
import {
  buildEditorExtensions,
  extractMentionRefs,
} from "@/components/doc/mention/editorExtensions";
import { publishChips } from "@/components/doc/mention/mentionChipsStore";
import { getAttachmentsKey } from "@/hooks/AttachmentHooks";
import { useResolveMentions } from "@/hooks/MentionHooks";
import type { AttachmentOwner } from "@/models/Attachment";
import type { MentionChipData } from "@/models/Mention";
import { cn } from "@/lib/utils";
import { isStructuredBody, parseBodyToJSON } from "@/utils/RichtextUtility";
import type { RelayCollabProvider } from "@/lib/collab/provider";

interface RichTextEditorProps {
  value: string;
  onChange: (json: string) => void;
  /** Reports the live heading outline on every edit that changes it (the doc's ToC). */
  onHeadingsChange?: (headings: DocHeading[]) => void;
  "aria-label"?: string;
  /** The doc or ticket that pasted, dropped, or picked files attach to; without it the body accepts no files. */
  attachTo?: AttachmentOwner;
  /** Binds to a live collab session (ws-25); body seeds only once the server confirms nothing to replay. */
  collab?: {
    doc: Y.Doc;
    provider: RelayCollabProvider;
    user: { name: string; color: string };
    serverReady: boolean;
    hasServerState: boolean;
  };
}

// Structured JSON bodies load as-is; anything else is legacy markdown.
const setBody = (editor: Editor, value: string) => {
  if (isStructuredBody(value)) {
    editor.commands.setContent(JSON.parse(value));
    return;
  }
  editor.commands.setContent(value, { contentType: "markdown" });
};

// Wires y-prosemirror's cursor plugin to session awareness so remote cursors render live.
const CollaborationCursors = Extension.create<{ awareness: Awareness }>({
  name: "collaborationCursors",
  addProseMirrorPlugins() {
    return [yCursorPlugin(this.options.awareness)];
  },
});

// No toolbar/box — formatting lives in the selection bubble; mention chips resolve from the store.
export const RichTextEditor = ({
  value,
  onChange,
  onHeadingsChange,
  "aria-label": ariaLabel,
  attachTo,
  collab,
}: RichTextEditorProps) => {
  const queryClient = useQueryClient();
  const extensionOptions = {
    attachTo: attachTo ?? null,
    onUploaded: () => void queryClient.invalidateQueries({ queryKey: [getAttachmentsKey] }),
  };
  // Tracks last applied value; only external changes re-apply, starting null to load initial value.
  const appliedValue = useRef<string | null>(null);
  const seeded = useRef(false);

  const editor = useEditor(
    {
      extensions: collab
        ? [
            ...buildEditorExtensions({ ...extensionOptions, collab: true }),
            Collaboration.configure({ fragment: collab.doc.getXmlFragment("prosemirror") }),
            CollaborationCursors.configure({ awareness: collab.provider.awareness }),
          ]
        : buildEditorExtensions(extensionOptions),
      immediatelyRender: false,
      editorProps: {
        attributes: {
          "aria-label": ariaLabel ?? "Doc body",
          "aria-multiline": "true",
          "data-placeholder": "Start writing…",
        },
      },
      onUpdate: ({ editor: e }) => {
        // Controlled re-application is skipped while a collab session owns the content.
        onChange(JSON.stringify(e.getJSON()));
      },
    },
    // Editor is recreated when the collab binding changes (a fresh session after a conflict reset).
    [collab ? collab.doc : "plain"],
  );

  // Apply the initial / external value: structured JSON, or legacy markdown.
  useEffect(() => {
    if (!editor || collab) return;
    if (appliedValue.current === value) return;
    appliedValue.current = value;
    setBody(editor, value);
  }, [editor, collab, value]);

  // Seeds only after the server confirms nothing to replay — earlier would push stale content into the CRDT.
  useEffect(() => {
    if (!editor || !collab || seeded.current) return;
    if (!collab.serverReady) return;
    if (collab.hasServerState) {
      seeded.current = true;
      return;
    }
    if (value === "") return;
    seeded.current = true;
    setBody(editor, value);
  }, [editor, collab, value]);

  // Resolves mention refs in one batched query so chips stay live while editing.
  const refs = useMemo(() => extractMentionRefs(parseBodyToJSON(value)), [value]);
  const { data } = useResolveMentions(refs);

  useEffect(() => {
    const map = new Map<string, MentionChipData>();
    for (const chip of data ?? []) map.set(`${chip.type}:${chip.id}`, chip);
    publishChips(map);
    return () => publishChips(new Map());
  }, [data]);

  const { isEmpty } = useEditorState({
    editor,
    selector: (ctx) => ({
      isEmpty: ctx.editor ? ctx.editor.state.doc.textContent.trim() === "" : true,
    }),
  }) ?? { isEmpty: true };

  // Recomputed every transaction so the ToC tracks the live page, not the load-time snapshot.
  const { headings, headingsKey } = useEditorState({
    editor,
    selector: (ctx) => {
      const headings = ctx.editor ? extractHeadingsFromJSON(ctx.editor.getJSON()) : [];
      return { headings, headingsKey: JSON.stringify(headings) };
    },
  }) ?? { headings: [], headingsKey: "[]" };

  const lastReportedHeadingsKey = useRef<string | null>(null);
  useEffect(() => {
    if (!onHeadingsChange) return;
    if (lastReportedHeadingsKey.current === headingsKey) return;
    lastReportedHeadingsKey.current = headingsKey;
    onHeadingsChange(headings);
  }, [headings, headingsKey, onHeadingsChange]);

  return (
    <div
      className={cn(
        "doc-editor doc-body-view prose-rich relative",
        isEmpty && "is-empty",
      )}
      data-testid="rich-text-editor"
    >
      {isEmpty && (
        <span
          aria-hidden="true"
          className="pointer-events-none absolute left-0 top-0 select-none text-muted-foreground/70"
        >
          Start writing…
        </span>
      )}
      <EditorContent editor={editor} />
      {editor && <FormatBubbleMenu editor={editor} />}
    </div>
  );
};
