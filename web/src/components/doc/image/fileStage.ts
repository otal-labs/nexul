import type { JSONContent } from "@tiptap/core";
import { toast } from "sonner";

import { errorMessage } from "@/api/client";
import { uploadAttachment } from "@/hooks/AttachmentHooks";
import { attachmentPath, uploadName, type AttachmentOwner } from "@/models/Attachment";
import { parseBodyToJSON } from "@/utils/RichtextUtility";

// Files dropped into a body whose entity doesn't exist yet, keyed by the object URL the editor shows them from.
export interface FileStage {
  add: (file: File) => string;
  entries: () => [string, File][];
  clear: () => void;
}

export const createFileStage = (): FileStage => {
  const staged = new Map<string, File>();
  return {
    add: (file) => {
      const url = URL.createObjectURL(file);
      staged.set(url, file);
      return url;
    },
    entries: () => [...staged],
    clear: () => {
      staged.forEach((_, url) => URL.revokeObjectURL(url));
      staged.clear();
    },
  };
};

const linkHref = (node: JSONContent): unknown => node.marks?.find((mark) => mark.type === "link")?.attrs?.href;

// Points an image or link node at its stored path, or drops it (null) when the file never made it.
const rewriteNode = (node: JSONContent, paths: Map<string, string | null>): JSONContent | null => {
  const url = node.type === "image" ? node.attrs?.src : linkHref(node);
  if (typeof url === "string" && paths.has(url)) {
    const path = paths.get(url);
    if (!path) return null;
    if (node.type === "image") return { ...node, attrs: { ...node.attrs, src: path } };
    return { ...node, marks: (node.marks ?? []).map((mark) => (mark.type === "link" ? { ...mark, attrs: { ...mark.attrs, href: path } } : mark)) };
  }
  if (!node.content) return node;
  return { ...node, content: node.content.map((child) => rewriteNode(child, paths)).filter((child) => child !== null) };
};

const rewriteBody = (body: string, paths: Map<string, string | null>): string =>
  JSON.stringify(rewriteNode(parseBodyToJSON(body), paths));

const stagedIn = (stage: FileStage, body: string) => stage.entries().filter(([url]) => body.includes(url));

const withoutStagedFiles = (stage: FileStage, body: string): string => {
  const staged = stagedIn(stage, body);
  if (staged.length === 0) return body;
  return rewriteBody(body, new Map(staged.map(([url]) => [url, null])));
};

// Uploads the staged files still in the body; returns the body with their stored paths, or null when none landed.
const attachStagedFiles = async (stage: FileStage, owner: AttachmentOwner, body: string): Promise<string | null> => {
  const paths = new Map<string, string | null>();
  for (const [url, file] of stagedIn(stage, body)) {
    try {
      paths.set(url, attachmentPath((await uploadAttachment(owner, file, uploadName(file))).id));
    } catch (error) {
      paths.set(url, null);
      toast.error(`Couldn't attach ${uploadName(file)}: ${errorMessage(error)}`);
    }
  }
  if (![...paths.values()].some(Boolean)) return null;
  return rewriteBody(body, paths);
};

// Files attach only to an entity that exists: create it without them, upload, then save the body with their paths.
export const createWithStagedFiles = async <T>(
  stage: FileStage,
  body: string,
  steps: {
    create: (body: string) => Promise<T>;
    ownerOf: (created: T) => AttachmentOwner;
    save: (created: T, body: string) => Promise<unknown>;
  },
): Promise<T> => {
  const created = await steps.create(withoutStagedFiles(stage, body));
  const attached = await attachStagedFiles(stage, steps.ownerOf(created), body);
  stage.clear();
  // The save's own hook already toasts a failure; the entity exists, so the dialog must not offer Create again.
  if (attached !== null) await steps.save(created, attached).catch(() => undefined);
  return created;
};
