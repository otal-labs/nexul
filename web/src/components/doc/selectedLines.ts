import type { CommandProps } from "@tiptap/core";
import type { ResolvedPos } from "@tiptap/pm/model";
import { TextSelection } from "@tiptap/pm/state";

const lineBreakAfter = ($pos: ResolvedPos): number | null => {
  if (!$pos.parent.isTextblock) return null;
  let found: number | null = null;
  $pos.parent.forEach((child, offset) => {
    const pos = $pos.start() + offset;
    if (found === null && child.type.name === "hardBreak" && pos >= $pos.pos) found = pos;
  });
  return found;
};

const lineBreakBefore = ($pos: ResolvedPos): number | null => {
  if (!$pos.parent.isTextblock) return null;
  let found: number | null = null;
  $pos.parent.forEach((child, offset) => {
    const pos = $pos.start() + offset;
    if (child.type.name === "hardBreak" && pos < $pos.pos) found = pos;
  });
  return found;
};

// Line breaks look like paragraphs, so block formatting would otherwise reach lines nobody selected.
export const isolateSelectedLines = ({ tr, dispatch }: CommandProps) => {
  if (!dispatch) return true;
  const { from, to, $from, $to } = tr.selection;
  const after = lineBreakAfter($to);
  const before = lineBreakBefore($from);
  if (after === null && before === null) return true;
  const steps = tr.steps.length;
  if (after !== null) tr.delete(after, after + 1).split(after);
  if (before !== null) tr.delete(before, before + 1).split(before);
  // Default mapping would carry a selection edge across the new split into the neighbouring line.
  const map = tr.mapping.slice(steps);
  tr.setSelection(TextSelection.create(tr.doc, map.map(from, 1), map.map(to, -1)));
  return true;
};
