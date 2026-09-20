import { Extension, type Editor } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import { uploadAttachment } from "@/hooks/AttachmentHooks";
import { attachmentPath, isInlineImage, type AttachmentOwner } from "@/models/Attachment";

export interface AttachmentUploadOptions {
  /** Owner uploads attach to; null disables paste/drop/pick (read-only bodies, entity-less forms). */
  owner: AttachmentOwner | null;
  /** Fired after each batch lands so the page's attachment list can refetch. */
  onUploaded?: () => void;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    attachmentUpload: {
      /** Uploads files against the owner; inserts an image node, or a link for non-images, at pos or the caret. */
      insertFiles: (files: File[], pos?: number) => ReturnType;
      /** Opens the browser's file picker for images and inserts whatever is chosen. */
      pickImage: () => ReturnType;
    };
  }
}

const filesFrom = (transfer: DataTransfer | null): File[] => Array.from(transfer?.files ?? []);

async function insertFiles(editor: Editor, options: AttachmentUploadOptions, files: File[], pos?: number) {
  if (!options.owner) return;
  for (const file of files) {
    try {
      const attachment = await uploadAttachment(options.owner, file, file.name || "Pasted image.png");
      const href = attachmentPath(attachment.id);
      const content = isInlineImage(attachment.content_type)
        ? { type: "image", attrs: { src: href, alt: attachment.name } }
        : { type: "text", text: attachment.name, marks: [{ type: "link", attrs: { href } }] };
      editor
        .chain()
        .focus()
        .insertContentAt(pos ?? editor.state.selection.to, content)
        .run();
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }
  options.onUploaded?.();
}

// The one place files enter a body — paste/drop/picker all upload first, never inline base64.
export const AttachmentUpload = Extension.create<AttachmentUploadOptions>({
  name: "attachmentUpload",

  addOptions() {
    return { owner: null };
  },

  addCommands() {
    return {
      insertFiles:
        (files, pos) =>
        ({ editor }) => {
          if (!this.options.owner || files.length === 0) return false;
          void insertFiles(editor, this.options, files, pos);
          return true;
        },
      pickImage:
        () =>
        ({ editor }) => {
          if (!this.options.owner) return false;
          const input = document.createElement("input");
          input.type = "file";
          input.accept = "image/*";
          input.multiple = true;
          input.addEventListener("change", () => {
            void insertFiles(editor, this.options, Array.from(input.files ?? []));
          });
          input.click();
          return true;
        },
    };
  },

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("attachment-upload"),
        props: {
          // ProseMirror re-runs an unhandled paste on a timer, which can outlive the editor.
          handlePaste: (_view, event) => {
            const files = filesFrom(event.clipboardData);
            if (files.length === 0 || this.editor.isDestroyed) return false;
            return this.editor.commands.insertFiles(files);
          },
          handleDrop: (view, event, _slice, moved) => {
            if (moved) return false;
            const files = filesFrom(event.dataTransfer);
            if (files.length === 0 || this.editor.isDestroyed) return false;
            const pos = view.posAtCoords({ left: event.clientX, top: event.clientY })?.pos;
            return this.editor.commands.insertFiles(files, pos);
          },
        },
      }),
    ];
  },
});

// True only when the editor has an owner to upload against — hides the Image entry otherwise.
export const canUploadAttachments = (editor: Editor): boolean => {
  const ext = editor.extensionManager.extensions.find((e) => e.name === AttachmentUpload.name);
  return !!(ext?.options as AttachmentUploadOptions | undefined)?.owner;
};
