import type { Editor } from "@tiptap/core";
import { NodeSelection } from "@tiptap/pm/state";
import { BubbleMenu } from "@tiptap/react/menus";
import {
  BoldIcon,
  Code2Icon,
  Heading1Icon,
  Heading2Icon,
  Heading3Icon,
  ItalicIcon,
  LinkIcon,
  ListIcon,
  ListOrderedIcon,
  QuoteIcon,
  StrikethroughIcon,
  TerminalIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useEditorState } from "@tiptap/react";

interface FormatBubbleMenuProps {
  editor: Editor;
}

interface BubbleButtonProps {
  label: string;
  active?: boolean;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}

const BubbleButton = ({ label, active, disabled, onClick, children }: BubbleButtonProps) => (
  <Button
    type="button"
    variant="ghost"
    size="sm"
    aria-label={label}
    aria-pressed={active}
    title={label}
    disabled={disabled}
    onClick={onClick}
    className={cn(
      "h-7 w-7 rounded-md p-0 text-foreground/80 hover:bg-white/10 hover:text-foreground",
      active && "bg-primary/15 text-primary hover:bg-primary/15 hover:text-primary",
    )}
  >
    {children}
  </Button>
);

const Divider = () => <span className="mx-0.5 h-4 w-px bg-white/15" aria-hidden="true" />;

// Floating bar over the selection since the page itself is the editor — no toolbar needed.
export const FormatBubbleMenu = ({ editor }: FormatBubbleMenuProps) => {
  const state = useEditorState({
    editor,
    selector: (ctx) => {
      const e = ctx.editor;
      return {
        bold: e.isActive("bold"),
        italic: e.isActive("italic"),
        strike: e.isActive("strike"),
        code: e.isActive("code"),
        h1: e.isActive("heading", { level: 1 }),
        h2: e.isActive("heading", { level: 2 }),
        h3: e.isActive("heading", { level: 3 }),
        bulletList: e.isActive("bulletList"),
        orderedList: e.isActive("orderedList"),
        blockquote: e.isActive("blockquote"),
        codeBlock: e.isActive("codeBlock"),
      };
    },
  });

  return (
    <BubbleMenu
      editor={editor}
      className="format-bubble"
      options={{ placement: "top", offset: 8 }}
      shouldShow={({ editor: e }) => {
        const { selection } = e.state;
        return !selection.empty && !(selection instanceof NodeSelection);
      }}
    >
      <BubbleButton
        label="Bold"
        active={state.bold}
        onClick={() => editor.chain().focus().toggleBold().run()}
      >
        <BoldIcon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Italic"
        active={state.italic}
        onClick={() => editor.chain().focus().toggleItalic().run()}
      >
        <ItalicIcon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Strikethrough"
        active={state.strike}
        onClick={() => editor.chain().focus().toggleStrike().run()}
      >
        <StrikethroughIcon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Inline code"
        active={state.code}
        onClick={() => editor.chain().focus().toggleCode().run()}
      >
        <Code2Icon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Insert link"
        onClick={() => {
          const href = window.prompt("Link URL");
          if (href) editor.chain().focus().setLink({ href }).run();
        }}
      >
        <LinkIcon className="size-3.5" />
      </BubbleButton>
      <Divider />
      <BubbleButton
        label="Heading 1"
        active={state.h1}
        onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
      >
        <Heading1Icon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Heading 2"
        active={state.h2}
        onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
      >
        <Heading2Icon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Heading 3"
        active={state.h3}
        onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
      >
        <Heading3Icon className="size-3.5" />
      </BubbleButton>
      <Divider />
      <BubbleButton
        label="Bullet list"
        active={state.bulletList}
        onClick={() => editor.chain().focus().toggleBulletList().run()}
      >
        <ListIcon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Ordered list"
        active={state.orderedList}
        onClick={() => editor.chain().focus().toggleOrderedList().run()}
      >
        <ListOrderedIcon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Blockquote"
        active={state.blockquote}
        onClick={() => editor.chain().focus().toggleBlockquote().run()}
      >
        <QuoteIcon className="size-3.5" />
      </BubbleButton>
      <BubbleButton
        label="Code block"
        active={state.codeBlock}
        onClick={() => editor.chain().focus().toggleCodeBlock().run()}
      >
        <TerminalIcon className="size-3.5" />
      </BubbleButton>
    </BubbleMenu>
  );
};
