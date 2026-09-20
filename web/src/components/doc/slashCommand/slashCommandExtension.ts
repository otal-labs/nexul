import { Extension } from "@tiptap/core";
import { ReactRenderer } from "@tiptap/react";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import type { EditorView } from "@tiptap/pm/view";
import Suggestion, { type SuggestionKeyDownProps, type SuggestionProps } from "@tiptap/suggestion";

import { canUploadAttachments } from "@/components/doc/image/attachmentUpload";
import { SlashCommandMenu, type SlashCommandMenuRef } from "@/components/doc/slashCommand/SlashCommandMenu";
import { filterSlashCommands, type SlashCommandItem } from "@/components/doc/slashCommand/slashCommands";

const renderSlashMenu = () => {
  let component: ReactRenderer<SlashCommandMenuRef> | null = null;
  let unmount: (() => void) | null = null;

  return {
    onStart: (props: SuggestionProps<SlashCommandItem>) => {
      component = new ReactRenderer(SlashCommandMenu, { props, editor: props.editor });
      unmount = props.mount(component.element);
    },
    onUpdate: (props: SuggestionProps<SlashCommandItem>) => component?.updateProps(props),
    onKeyDown: (props: SuggestionKeyDownProps) => component?.ref?.onKeyDown(props) ?? false,
    onExit: () => {
      unmount?.();
      component?.destroy();
    },
  };
};

// "/" trigger opens the same block-insert menu as the gutter's PlusMenuExtension button.
export const SlashCommandExtension = Extension.create({
  name: "slashCommand",

  addProseMirrorPlugins() {
    return [
      Suggestion({
        editor: this.editor,
        char: "/",
        pluginKey: new PluginKey("slash-command"),
        items: ({ query, editor }) => filterSlashCommands(query, canUploadAttachments(editor)),
        command: ({ editor, range, props }) => (props as SlashCommandItem).run(editor, range),
        render: renderSlashMenu,
      }),
    ];
  },
});

function isEmptyParagraph(view: EditorView): boolean {
  const { $from } = view.state.selection;
  return $from.parent.type.name === "paragraph" && $from.parent.content.size === 0;
}

// Only shows next to the current empty paragraph (no mousemove tracking); click just types "/".
export const PlusMenuExtension = Extension.create({
  name: "plusMenu",

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("plus-menu"),
        view: (view) => {
          const parent = view.dom.parentElement;
          if (!parent) return {};
          if (getComputedStyle(parent).position === "static") {
            parent.style.position = "relative";
          }

          const button = document.createElement("button");
          button.type = "button";
          button.className = "plus-menu-button";
          button.setAttribute("aria-label", "Insert block");
          button.textContent = "+";
          button.style.display = "none";
          button.addEventListener("mousedown", (event) => {
            event.preventDefault();
            view.dispatch(view.state.tr.insertText("/"));
            view.focus();
          });
          parent.appendChild(button);

          const update = (view: EditorView): void => {
            if (!view.editable || !isEmptyParagraph(view)) {
              button.style.display = "none";
              return;
            }
            const pos = view.state.selection.$from.before();
            const coords = view.coordsAtPos(pos);
            const parentRect = parent.getBoundingClientRect();
            button.style.display = "flex";
            button.style.top = `${coords.top - parentRect.top}px`;
            button.style.left = `${coords.left - parentRect.left - 28}px`;
          };

          update(view);

          return {
            update,
            destroy: () => button.remove(),
          };
        },
      }),
    ];
  },
});
