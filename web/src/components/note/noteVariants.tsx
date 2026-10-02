import { useSearchParams } from "react-router";

// Prototype only: the three candidate note pills and dialogs, picked with ?note=a|b|c; ?reader=1 views as a reader.
export interface NoteVariant {
  key: "a" | "b" | "c";
  label: string;
  pill: "file" | "preview" | "title";
  dialog: "centred" | "sheet" | "tall";
  editing: "always" | "switch";
}

export const NOTE_VARIANTS: Record<NoteVariant["key"], NoteVariant> = {
  a: {
    key: "a",
    label: "A: file pill under the line, centred dialog the size of the body card, always editable",
    pill: "file",
    dialog: "centred",
    editing: "always",
  },
  b: {
    key: "b",
    label: "B: preview card with the first lines, right-side sheet, read view with an Edit switch",
    pill: "preview",
    dialog: "sheet",
    editing: "switch",
  },
  c: {
    key: "c",
    label: "C: the line is the link with a small file chip, tall near-full-screen dialog, always editable",
    pill: "title",
    dialog: "tall",
    editing: "always",
  },
};

export const useNoteVariant = (): { variant: NoteVariant; reader: boolean } => {
  const [params] = useSearchParams();
  const key = params.get("note");
  const variant = key === "b" || key === "c" ? NOTE_VARIANTS[key] : NOTE_VARIANTS.a;
  return { variant, reader: params.get("reader") === "1" };
};

// The first two lines of prose in a note's markdown, for the preview card.
export const previewLines = (markdown: string): string[] =>
  markdown
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "" && !line.startsWith("#") && !line.startsWith("!["))
    .slice(0, 2)
    .map((line) => line.replace(/^([-*]|\d+\.)\s+/, "").replace(/[*_`]/g, ""));
