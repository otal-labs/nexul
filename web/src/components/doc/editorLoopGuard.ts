import { Extension } from "@tiptap/core";
import { Plugin } from "@tiptap/pm/state";

import { enqueue } from "@/lib/browserLogs";

// Mutation batches with no task between them: only a loop of the editor redrawing what something else rewrites gets here.
export const loopBatches = 300;
const keptRecords = 200;
const topLines = 5;

const tag = (node: Node | null) => {
  if (!(node instanceof Element)) return node?.nodeName ?? "?";
  const cls = node.getAttribute("class")?.trim().split(/\s+/).join(".");
  return cls ? `${node.nodeName}.${cls}` : node.nodeName;
};

const short = (value: string | null | undefined) => (value ?? "").slice(0, 80);

const describeRecord = (m: MutationRecord) => {
  if (m.type === "attributes") return `attr ${m.attributeName} on <${tag(m.target)}> ${short(m.oldValue)} -> ${short((m.target as Element).getAttribute(m.attributeName ?? ""))}`;
  if (m.type === "characterData") return `text in <${tag(m.target.parentNode)}>`;
  return `children of <${tag(m.target)}> +[${[...m.addedNodes].map(tag).join(",")}] -[${[...m.removedNodes].map(tag).join(",")}]`;
};

const reportLoop = (records: MutationRecord[]) => {
  const counts = new Map<string, number>();
  for (const line of records.map(describeRecord)) counts.set(line, (counts.get(line) ?? 0) + 1);
  const top = [...counts].sort((a, b) => b[1] - a[1]).slice(0, topLines);
  enqueue("error", "doc editor mutation loop", {
    // An extension that rewrites pages marks the root element, so its attribute names say which one ran.
    html_attrs: document.documentElement.getAttributeNames().join(" "),
    ...Object.fromEntries(top.map(([line, count], i) => [`mutation_${i + 1}`, `${count}x ${line}`])),
  });
};

// A read-only editor stops reading the page back, which ends a loop with a page-rewriting extension until the next task.
export const EditorLoopGuard = Extension.create({
  name: "editorLoopGuard",
  addProseMirrorPlugins() {
    const { editor } = this;
    return [
      new Plugin({
        view: (view) => {
          const records: MutationRecord[] = [];
          let batches = 0;
          let reported = false;
          let resetQueued = false;
          const observer = new MutationObserver((batch) => {
            records.push(...batch);
            if (records.length > keptRecords) records.splice(0, records.length - keptRecords);
            if (!resetQueued) {
              resetQueued = true;
              setTimeout(() => {
                batches = 0;
                resetQueued = false;
              });
            }
            batches += 1;
            if (batches !== loopBatches || !editor.isEditable) return;
            if (!reported) reportLoop(records);
            reported = true;
            editor.setEditable(false, false);
            setTimeout(() => !editor.isDestroyed && editor.setEditable(true, false));
          });
          observer.observe(view.dom, { subtree: true, childList: true, attributes: true, attributeOldValue: true, characterData: true });
          return { destroy: () => observer.disconnect() };
        },
      }),
    ];
  },
});
