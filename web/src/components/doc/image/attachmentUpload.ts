import { Extension, type ChainedCommands, type Editor } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import type { FileStage } from "@/components/doc/image/fileStage";
import { uploadAttachment } from "@/hooks/AttachmentHooks";
import { attachmentPath, isInlineImage, uploadName, type AttachmentOwner } from "@/models/Attachment";

export interface AttachmentUploadOptions {
  /** Owner uploads attach to; with neither an owner nor a stage, paste/drop/pick is off (read-only bodies). */
  owner: AttachmentOwner | null;
  /** A create form's holding area: files show from a local URL and upload once the entity exists. */
  stage: FileStage | null;
  /** Fired after each batch lands so the page's attachment list can refetch. */
  onUploaded?: () => void;
}

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    attachmentUpload: {
      /** Uploads files against the owner, or stages them; inserts an image node, or a link for non-images, at pos or the caret. */
      insertFiles: (files: File[], pos?: number) => ReturnType;
      /** Opens the browser's file picker for images and inserts whatever is chosen. */
      pickImage: () => ReturnType;
    };
  }
}

const filesFrom = (transfer: DataTransfer | null): File[] => Array.from(transfer?.files ?? []);

const insertFile = (chain: () => ChainedCommands, at: number, name: string, contentType: string, href: string) => {
  const content = isInlineImage(contentType)
    ? { type: "image", attrs: { src: href, alt: name } }
    : { type: "text", text: name, marks: [{ type: "link", attrs: { href } }] };
  chain().focus().insertContentAt(at, content).run();
};

async function uploadFiles(editor: Editor, owner: AttachmentOwner, options: AttachmentUploadOptions, files: File[], pos?: number) {
  for (const file of files) {
    try {
      const attachment = await uploadAttachment(owner, file, uploadName(file));
      const at = pos ?? editor.state.selection.to;
      insertFile(() => editor.chain(), at, attachment.name, attachment.content_type, attachmentPath(attachment.id));
    } catch (error) {
      toast.error(errorMessage(error));
    }
  }
  options.onUploaded?.();
}

const acceptsFiles = (options: AttachmentUploadOptions): boolean => !!(options.owner || options.stage);

// The one place files enter a body — paste/drop/picker upload (or stage) first, never inline base64.
export const AttachmentUpload = Extension.create<AttachmentUploadOptions>({
  name: "attachmentUpload",

  addOptions() {
    return { owner: null, stage: null };
  },

  addCommands() {
    return {
      insertFiles:
        (files, pos) =>
        ({ editor, chain, tr }) => {
          const { owner, stage } = this.options;
          if (files.length === 0) return false;
          if (owner) {
            void uploadFiles(editor, owner, this.options, files, pos);
            return true;
          }
          if (!stage) return false;
          // Staging is synchronous, so it inserts through this command's own transaction.
          for (const file of files) insertFile(chain, pos ?? tr.selection.to, uploadName(file), file.type, stage.add(file));
          return true;
        },
      pickImage:
        () =>
        ({ editor }) => {
          if (!acceptsFiles(this.options)) return false;
          const input = document.createElement("input");
          input.type = "file";
          input.accept = "image/*";
          input.multiple = true;
          input.addEventListener("change", () => editor.commands.insertFiles(Array.from(input.files ?? [])));
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

// True only when the editor has an owner or a stage to take files — hides the Image entry otherwise.
export const canUploadAttachments = (editor: Editor): boolean => {
  const ext = editor.extensionManager.extensions.find((e) => e.name === AttachmentUpload.name);
  return !!ext && acceptsFiles(ext.options as AttachmentUploadOptions);
};
