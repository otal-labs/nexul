import type { Editor, Range } from "@tiptap/core";
import {
  Heading1Icon,
  Heading2Icon,
  Heading3Icon,
  ImageIcon,
  ListIcon,
  ListOrderedIcon,
  MinusIcon,
  QuoteIcon,
  TableIcon,
  TerminalIcon,
  TypeIcon,
} from "lucide-react";

import { isolateSelectedLines } from "@/components/doc/selectedLines";

export interface SlashCommandItem {
  id: string;
  label: string;
  icon: typeof TypeIcon;
  keywords: string[];
  /** Only offered when the editor has an attachment owner to upload against. */
  requiresUpload?: boolean;
  run: (editor: Editor, range?: Range) => void;
}

const chain = (editor: Editor, range?: Range) => {
  const c = editor.chain().focus();
  return range ? c.deleteRange(range) : c;
};

const lines = (editor: Editor, range?: Range) => chain(editor, range).command(isolateSelectedLines);

// Mirrors what the editor's nodes support — never promise a block type it can't produce.
export const slashCommandItems: SlashCommandItem[] = [
  {
    id: "text",
    label: "Text",
    icon: TypeIcon,
    keywords: ["paragraph", "text"],
    run: (editor, range) => lines(editor, range).setParagraph().run(),
  },
  {
    id: "h1",
    label: "Heading 1",
    icon: Heading1Icon,
    keywords: ["heading", "h1", "title"],
    run: (editor, range) => lines(editor, range).setHeading({ level: 1 }).run(),
  },
  {
    id: "h2",
    label: "Heading 2",
    icon: Heading2Icon,
    keywords: ["heading", "h2", "subtitle"],
    run: (editor, range) => lines(editor, range).setHeading({ level: 2 }).run(),
  },
  {
    id: "h3",
    label: "Heading 3",
    icon: Heading3Icon,
    keywords: ["heading", "h3"],
    run: (editor, range) => lines(editor, range).setHeading({ level: 3 }).run(),
  },
  {
    id: "bulletList",
    label: "Bullet list",
    icon: ListIcon,
    keywords: ["list", "bullet", "ul"],
    run: (editor, range) => lines(editor, range).toggleBulletList().run(),
  },
  {
    id: "orderedList",
    label: "Numbered list",
    icon: ListOrderedIcon,
    keywords: ["list", "ordered", "number", "ol"],
    run: (editor, range) => lines(editor, range).toggleOrderedList().run(),
  },
  {
    id: "blockquote",
    label: "Quote",
    icon: QuoteIcon,
    keywords: ["quote", "blockquote"],
    run: (editor, range) => lines(editor, range).toggleBlockquote().run(),
  },
  {
    id: "codeBlock",
    label: "Code block",
    icon: TerminalIcon,
    keywords: ["code"],
    run: (editor, range) => lines(editor, range).toggleCodeBlock().run(),
  },
  {
    id: "table",
    label: "Table",
    icon: TableIcon,
    keywords: ["table", "grid", "rows", "columns"],
    run: (editor, range) => chain(editor, range).insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run(),
  },
  {
    id: "image",
    label: "Image",
    icon: ImageIcon,
    keywords: ["image", "picture", "photo", "screenshot", "upload", "file"],
    requiresUpload: true,
    run: (editor, range) => chain(editor, range).pickImage().run(),
  },
  {
    id: "divider",
    label: "Divider",
    icon: MinusIcon,
    keywords: ["divider", "hr", "line", "separator"],
    run: (editor, range) => chain(editor, range).setHorizontalRule().run(),
  },
];

export function filterSlashCommands(query: string, canUpload = true): SlashCommandItem[] {
  const q = query.trim().toLowerCase();
  const available = canUpload ? slashCommandItems : slashCommandItems.filter((item) => !item.requiresUpload);
  if (!q) return available;
  return available.filter(
    (item) => item.label.toLowerCase().includes(q) || item.keywords.some((k) => k.includes(q)),
  );
}
