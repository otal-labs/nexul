import { Extension } from "@tiptap/core";
import type { EditorState, Transaction } from "@tiptap/pm/state";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { ySyncPluginKey } from "@tiptap/y-tiptap";

import type { MentionRef } from "@/models/Mention";
import { useWorkspaceStore } from "@/stores/workspaceStore";

// A link to a ticket or doc page in this workspace names a reference; boards, settings, and anything with a query or anchor stay links.
export const recordRefFromHref = (href: string, origin: string, workspaceSlug: string): MentionRef | null => {
  if (!URL.canParse(href, origin)) return null;
  const url = new URL(href, origin);
  if (url.origin !== origin || url.search || url.hash) return null;
  const [workspace, section, ...rest] = url.pathname.split("/").filter((s) => s !== "");
  if (!workspaceSlug || workspace !== workspaceSlug) return null;
  const id = rest.at(-1);
  if (section === "tickets" && rest.length === 1 && id) return { type: "ticket", id };
  if (section === "docs" && rest.length <= 2 && id) return { type: "doc", id };
  return null;
};

export const currentRecordRef = (href: unknown): MentionRef | null =>
  typeof href === "string"
    ? recordRefFromHref(href, window.location.origin, useWorkspaceStore.getState().selectedWorkspaceSlug)
    : null;

interface LinkRun {
  from: number;
  to: number;
  href: string;
  text: string;
  ref: MentionRef;
}

const recordLinkRuns = (state: EditorState): LinkRun[] => {
  const link = state.schema.marks.link;
  const runs: LinkRun[] = [];
  if (!link) return runs;
  state.doc.descendants((node, pos) => {
    if (!node.isText) return;
    const href: unknown = link.isInSet(node.marks)?.attrs.href;
    const ref = currentRecordRef(href);
    if (!ref || typeof href !== "string") return;
    const last = runs.at(-1);
    if (last && last.to === pos && last.href === href) {
      last.to = pos + node.nodeSize;
      last.text += node.text ?? "";
      return;
    }
    runs.push({ from: pos, to: pos + node.nodeSize, href, text: node.text ?? "", ref });
  });
  return runs;
};

// Replaces every record link with a mention node, last first so earlier positions stay valid.
const linksToMentions = (state: EditorState): Transaction | null => {
  const mention = state.schema.nodes.mention;
  const runs = recordLinkRuns(state);
  if (!mention || runs.length === 0) return null;
  const tr = state.tr;
  for (const run of runs.reverse()) {
    const label = /^https?:\/\//.test(run.text) ? run.ref.id : run.text;
    tr.replaceWith(run.from, run.to, mention.create({ type: run.ref.type, id: run.ref.id, label }));
  }
  return tr;
};

const isRemote = (tr: Transaction) => (tr.getMeta(ySyncPluginKey) as { isChangeOrigin?: boolean } | undefined)?.isChangeOrigin === true;

// A pasted, autolinked, or loaded record URL becomes the same pill the @ picker inserts; remote changes are left to the collaborator who made them.
export const RecordLinkMentions = Extension.create({
  name: "recordLinkMentions",
  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("recordLinkMentions"),
        appendTransaction: (transactions, _oldState, state) => {
          if (!transactions.some((tr) => tr.docChanged) || transactions.some(isRemote)) return null;
          return linksToMentions(state);
        },
      }),
    ];
  },
  onCreate() {
    const tr = linksToMentions(this.editor.state);
    if (!tr) return;
    this.editor.view.dispatch(tr.setMeta("preventUpdate", true).setMeta("addToHistory", false));
  },
});
