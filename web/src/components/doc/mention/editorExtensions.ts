import StarterKit from "@tiptap/starter-kit";
import { Markdown } from "@tiptap/markdown";
import Mention from "@tiptap/extension-mention";
import CodeBlockLowlight from "@tiptap/extension-code-block-lowlight";
import Image from "@tiptap/extension-image";
import { ReactNodeViewRenderer, ReactRenderer } from "@tiptap/react";
import type { SuggestionKeyDownProps, SuggestionProps } from "@tiptap/suggestion";
import type { JSONContent } from "@tiptap/core";

import { CodeBlockView } from "@/components/doc/codeBlock/CodeBlockView";
import { AttachmentImageView } from "@/components/doc/image/AttachmentImageView";
import { AttachmentUpload, type AttachmentUploadOptions } from "@/components/doc/image/attachmentUpload";
import { lowlight } from "@/components/doc/codeBlock/lowlight";
import {
  MentionSuggestions,
  type MentionSuggestionsRef,
} from "@/components/doc/mention/MentionSuggestions";
import { MentionNodeView } from "@/components/doc/mention/MentionNodeView";
import { PlusMenuExtension, SlashCommandExtension } from "@/components/doc/slashCommand/slashCommandExtension";
import { searchMentions } from "@/hooks/MentionHooks";
import type { MentionRef, MentionSearchResult } from "@/models/Mention";

// Matches a canonical mention link: the URL is the identifier, the label is just display text.
const mentionLinkRe = /^\[([^\]]+)\]\((\/(?:tickets|docs|people)\/)([a-zA-Z0-9-]+)\)/;

const mentionPaths: Record<string, string> = { ticket: "/tickets/", doc: "/docs/", person: "/people/" };
const mentionTypesByPath: Record<string, string> = { "/tickets/": "ticket", "/docs/": "doc", "/people/": "person" };

// Must stay in sync with the Go converter (internal/docs/richtext/mention.go).
function mentionHref(type: string, id: string): string {
  return `${mentionPaths[type] ?? "/"}${id}`;
}

// A person mention's label is the bare login, written as @login in markdown.
const mentionMarkdownLabel = (type: string, label: string) => (type === "person" ? `@${label}` : label);

function escapeMarkdownLabel(label: string): string {
  return label.replace(/([\\[\]*_`~])/g, "\\$1");
}

// Collects unique refs so a whole render resolves in one batched request.
export function extractMentionRefs(doc: JSONContent | null): MentionRef[] {
  const refs: MentionRef[] = [];
  const seen = new Set<string>();
  const visit = (node: JSONContent | null | undefined) => {
    if (!node) return;
    if (node.type === "mention") {
      const type = node.attrs?.type;
      const id = node.attrs?.id;
      if ((type === "ticket" || type === "doc") && typeof id === "string" && id) {
        const key = `${type}:${id}`;
        if (!seen.has(key)) {
          seen.add(key);
          refs.push({ type, id });
        }
      }
    }
    for (const child of node.content ?? []) visit(child);
  };
  visit(doc);
  return refs;
}

// Mounts the @-picker into the suggestion plugin's floating element; mount() owns positioning.
const renderSuggestions = () => {
  let component: ReactRenderer<MentionSuggestionsRef> | null = null;
  let unmount: (() => void) | null = null;

  return {
    onStart: (props: SuggestionProps<MentionSearchResult>) => {
      component = new ReactRenderer(MentionSuggestions, { props, editor: props.editor });
      unmount = props.mount(component.element);
    },
    onUpdate: (props: SuggestionProps<MentionSearchResult>) => component?.updateProps(props),
    onKeyDown: (props: SuggestionKeyDownProps) => component?.ref?.onKeyDown(props) ?? false,
    onExit: () => {
      unmount?.();
      component?.destroy();
    },
  };
};

// Shared by the editor and markdown/HTML conversion so mention nodes behave identically everywhere.
export interface EditorExtensionOptions {
  collab?: boolean;
  /** The doc or ticket pasted/dropped files attach to; omitted for read-only bodies. */
  attachTo?: AttachmentUploadOptions["owner"];
  onUploaded?: AttachmentUploadOptions["onUploaded"];
  /** The gutter "+" beside an empty line; "/" still opens the same menu without it. */
  plusMenu?: boolean;
}

export function buildEditorExtensions({
  collab = false,
  attachTo = null,
  onUploaded,
  plusMenu = true,
}: EditorExtensionOptions = {}) {
  return [
    // Collab mode's undo comes from the Yjs Collaboration extension; StarterKit's UndoRedo must stand down for Mod-Z.
    StarterKit.configure({ codeBlock: false, ...(collab && { undoRedo: false }) }),
    CodeBlockLowlight.extend({
      addNodeView() {
        return ReactNodeViewRenderer(CodeBlockView);
      },
    }).configure({ lowlight }),
    Markdown,
    Image.extend({
      addNodeView() {
        return ReactNodeViewRenderer(AttachmentImageView);
      },
    }),
    AttachmentUpload.configure({ owner: attachTo, ...(onUploaded ? { onUploaded } : {}) }),
    SlashCommandExtension,
    ...(plusMenu ? [PlusMenuExtension] : []),
    Mention.extend({
      markdownTokenizer: {
        name: "mention",
        level: "inline",
        start: (src: string) => {
          const match = src.match(/\[[^\]]*\]\((?:\/tickets\/|\/docs\/|\/people\/)/);
          return match ? (match.index ?? -1) : -1;
        },
        tokenize: (src: string) => {
          const match = mentionLinkRe.exec(src);
          if (!match) return undefined;
          const type = mentionTypesByPath[match[2] ?? ""] ?? "doc";
          const label = match[1] ?? "";
          return {
            type: "mention",
            raw: match[0],
            attributes: { type, id: match[3] ?? "", label: type === "person" ? label.replace(/^@/, "") : label },
          };
        },
      },
      parseMarkdown: (token, helpers) =>
        helpers.createNode("mention", token.attributes ?? {}),
      renderMarkdown: (node) => {
        const type = node.attrs?.type ?? "";
        const label = mentionMarkdownLabel(type, node.attrs?.label ?? node.attrs?.id ?? "");
        return `[${escapeMarkdownLabel(label)}](${mentionHref(type, node.attrs?.id ?? "")})`;
      },
      addAttributes() {
        return {
          ...this.parent?.(),
          type: {
            default: "ticket",
            parseHTML: (element: HTMLElement) =>
              element.getAttribute("data-mention-type") ?? "ticket",
            renderHTML: (attributes: Record<string, string>) => ({
              "data-mention-type": attributes.type,
            }),
          },
        };
      },
      parseHTML() {
        return [{ tag: "span[data-mention-type]" }];
      },
      addNodeView() {
        return ReactNodeViewRenderer(MentionNodeView);
      },
    }).configure({
      renderText: ({ node }) => `@${node.attrs.label ?? node.attrs.id}`,
      renderHTML: ({ node }) => [
        "span",
        {
          "data-mention-type": node.attrs.type,
          "data-mention-id": node.attrs.id,
          "data-mention-label": node.attrs.label ?? node.attrs.id,
          class: "mention-chip",
        },
        `@${node.attrs.label ?? node.attrs.id}`,
      ],
      suggestion: {
        char: "@",
        allowSpaces: false,
        startOfLine: false,
        debounce: 200,
        items: async ({ query }) => {
          if (!query.trim()) return [];
          return searchMentions(query, 8);
        },
        command: ({ editor, range, props }) => {
          const item = props as unknown as MentionSearchResult;
          // A person keeps their user id and login; the chip shows their live display name.
          const label = item.type === "person" ? (item.login ?? item.title) : item.title;
          editor
            .chain()
            .focus()
            .insertContentAt(range, [
              { type: "mention", attrs: { type: item.type, id: item.id, label } },
              { type: "text", text: " " },
            ])
            .run();
          editor.view.dom.ownerDocument.defaultView?.getSelection()?.collapseToEnd();
        },
        render: renderSuggestions,
      },
    }),
  ];
}
