import { Extension, type Editor, type JSONContent } from "@tiptap/core";
import type { Transaction } from "@tiptap/pm/state";
import Collaboration from "@tiptap/extension-collaboration";
import { useQueryClient } from "@tanstack/react-query";
import { EditorContent, useEditor, useEditorState } from "@tiptap/react";
import { yCursorPlugin, ySyncPluginKey } from "@tiptap/y-tiptap";
import { useEffect, useMemo, useRef } from "react";
import type { Awareness } from "y-protocols/awareness";
import type * as Y from "yjs";

import type { MentionChipData } from "@nexul/client-core/mention";

import { DiscordTimeLayer } from "@/components/doc/discordTime/DiscordTimeLayer";
import { FormatBubbleMenu } from "@/components/doc/FormatBubbleMenu";
import { TableBubbleMenu } from "@/components/doc/TableBubbleMenu";
import { extractHeadingsFromJSON, type DocHeading } from "@/components/doc/docHeadings";
import { upgradedList } from "@/components/doc/legacyChecklist";
import {
  buildEditorExtensions,
  extractMentionRefs,
} from "@/components/doc/mention/editorExtensions";
import type { FileStage } from "@/components/doc/image/fileStage";
import { publishChips } from "@/components/doc/mention/mentionChipsStore";
import { getAttachmentsKey } from "@/hooks/AttachmentHooks";
import { useResolveMentions } from "@/hooks/MentionHooks";
import type { AttachmentOwner } from "@/models/Attachment";
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
  /** A create dialog's holding area for files pasted before the entity exists; it uploads them after creating. */
  stage?: FileStage;
  /** A dialog's short body: no gutter "+", a short minimum height, and the ticket dialog's placeholder. */
  compact?: boolean;
  /** Binds to a live collab session; the body loads into it only when the server picks this editor to seed. */
  collab?: {
    doc: Y.Doc;
    provider: RelayCollabProvider;
    user: { name: string; color: string };
    seed: boolean;
  };
}

// Structured JSON bodies load as-is; anything else is legacy markdown.
const setBody = (editor: Editor, value: string, emitUpdate = true) => {
  if (isStructuredBody(value)) {
    editor.commands.setContent(parseBodyToJSON(value), { emitUpdate });
    return;
  }
  editor.commands.setContent(value, { contentType: "markdown", emitUpdate });
};

// Only a lone empty paragraph takes the hint; an empty list, quote, or code block sits where the hint would draw.
const isBlankDoc = (doc: Editor["state"]["doc"]) =>
  doc.childCount === 1 && doc.firstChild?.type.name === "paragraph" && doc.firstChild.content.size === 0;

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
  stage,
  compact = false,
  collab,
}: RichTextEditorProps) => {
  const queryClient = useQueryClient();
  const placeholder = compact ? "Add description…" : "Start writing…";
  const extensionOptions = {
    plusMenu: !compact,
    attachTo: attachTo ?? null,
    stage: stage ?? null,
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
          "data-placeholder": placeholder,
        },
      },
      onUpdate: ({ editor: e }) => {
        // Controlled re-application is skipped while a collab session owns the content.
        const json = JSON.stringify(e.getJSON());
        // The echo of this edit is not an external change; re-applying it would reset the cursor mid-typing.
        appliedValue.current = json;
        onChange(json);
      },
    },
    // Editor is recreated when the collab binding changes (a fresh session after a conflict reset).
    [collab ? collab.doc : "plain"],
  );

  // Applied silently: loading a body is not an edit, and its normalized echo would autosave an untouched record.
  useEffect(() => {
    if (!editor || collab) return;
    if (appliedValue.current === value) return;
    appliedValue.current = value;
    setBody(editor, value, false);
  }, [editor, collab, value]);

  // Only the editor the server picks seeds an empty room; two seeding it would show the body twice.
  useEffect(() => {
    if (!editor || !collab?.seed || seeded.current || value === "") return;
    seeded.current = true;
    collab.provider.seed(() => setBody(editor, value));
  }, [editor, collab, value]);

  // A room saved before task items holds "[ ] " bullets. Once its replay has landed, an editor alone in the room rewrites
  // them in one edit, deferred a tick so no replayed update is still to land inside an item it replaces. With peers
  // present nothing changes: their concurrent edits would be lost or doubled, so the next solo open converts it.
  useEffect(() => {
    if (!editor || !collab) return;
    const { awareness } = collab.provider;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const upgrade = () => {
      if (!editor.isEditable) return;
      const peerEdits = [...awareness.getStates()].some(([id, state]) => id !== collab.doc.clientID && state.user?.activity === "editing");
      if (peerEdits) return;
      const { doc, schema, tr } = editor.state;
      const edits: { from: number; to: number; next: JSONContent[] }[] = [];
      doc.descendants((node, pos) => {
        if (node.type.name !== "bulletList") return true;
        const next = upgradedList(node.toJSON());
        if (next) edits.push({ from: pos, to: pos + node.nodeSize, next });
        return false;
      });
      if (edits.length === 0) return;
      for (const { from, to, next } of edits.reverse()) tr.replaceWith(from, to, next.map((n) => schema.nodeFromJSON(n)));
      editor.view.dispatch(tr);
    };
    const schedule = () => {
      clearTimeout(timer);
      timer = setTimeout(upgrade, 0);
    };
    const onUpdate = ({ transaction }: { transaction: Transaction }) => {
      if (transaction.getMeta(ySyncPluginKey)?.isChangeOrigin) schedule();
    };
    schedule();
    editor.on("transaction", onUpdate);
    awareness.on("change", schedule);
    return () => {
      clearTimeout(timer);
      editor.off("transaction", onUpdate);
      awareness.off("change", schedule);
    };
  }, [editor, collab]);

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
      isEmpty: ctx.editor ? isBlankDoc(ctx.editor.state.doc) : true,
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
    // A joining editor is blank until its room syncs; the first outline reported is a real one, not that gap.
    if (lastReportedHeadingsKey.current === null && headings.length === 0) return;
    if (lastReportedHeadingsKey.current === headingsKey) return;
    lastReportedHeadingsKey.current = headingsKey;
    onHeadingsChange(headings);
  }, [headings, headingsKey, onHeadingsChange]);

  return (
    <div
      className={cn(
        "doc-editor doc-body-view prose-rich relative",
        compact && "doc-editor-compact text-sm",
        isEmpty && "is-empty",
      )}
      data-testid="rich-text-editor"
    >
      {isEmpty && (
        <span
          aria-hidden="true"
          className="pointer-events-none absolute left-0 top-0 select-none text-muted-foreground/70"
        >
          {placeholder}
        </span>
      )}
      <DiscordTimeLayer>
        <EditorContent editor={editor} />
      </DiscordTimeLayer>
      {editor && <FormatBubbleMenu editor={editor} />}
      {editor && <TableBubbleMenu editor={editor} />}
    </div>
  );
};
