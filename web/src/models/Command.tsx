import type { ComponentType } from "react";

export interface CommandItem {
  id: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  // Matched like the label, never shown: "palette colour" on a theme.
  keywords?: string;
  // Trails the label in mono: a ticket key, a project prefix, a pull request's repo#number.
  hint?: string;
  // What or where the row is, in plain text before the hint: "Doc", "Theme", a project's name.
  context?: string;
  // A ticket's status, drawn as its stage's dot beside the name.
  status?: { label: string; kind: string };
  // When the record last changed, shown as a short age.
  updatedAt?: string;
  // The value in use, such as the current theme.
  checked?: boolean;
  run: () => void;
}

export interface CommandGroup {
  heading: string;
  items: CommandItem[];
  // A run of the typed list; its heading names it for screen readers only.
  quiet?: boolean;
}

export const IN_TEXT = "In text";
const LEAD = 3;

export const commandOptionId = (index: number) => `command-option-${index}`;

// Every word typed has to appear in the label, keywords, context or hint; a label that starts with the query ranks first.
const rank = (item: CommandItem, words: string[]): number => {
  const label = item.label.toLowerCase();
  const haystack = `${label} ${item.keywords ?? ""} ${item.context ?? ""} ${item.hint ?? ""}`.toLowerCase();
  if (!words.every((word) => haystack.includes(word))) return -1;
  if (label.startsWith(words.join(" "))) return 2;
  if (label.split(/\s+/).some((part) => part.startsWith(words[0] ?? ""))) return 1;
  return 0;
};

// Browsing keeps the groups as they come. Typed: up to three local rows whose label leads with a typed word, the search's
// title hits by kind, every other local row, then In text, so a row under In text is always a body match.
export const arrangeCommandGroups = (local: CommandGroup[], found: CommandGroup[], query: string): CommandGroup[] => {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  if (words.length === 0) return [...found, ...local].filter((group) => group.items.length > 0);
  const ranked = local
    .flatMap((group) => group.items)
    .map((item) => ({ item, score: rank(item, words) }))
    .filter((entry) => entry.score >= 0)
    .sort((a, b) => b.score - a.score);
  const lead = ranked.filter((entry) => entry.score > 0).slice(0, LEAD);
  const rest = ranked.filter((entry) => !lead.includes(entry));
  const inText = found.filter((group) => group.heading === IN_TEXT);
  const titled = found.filter((group) => group.heading !== IN_TEXT).map((group) => ({ ...group, quiet: true }));
  return [
    { heading: "Top matches", items: lead.map((entry) => entry.item), quiet: true },
    ...titled,
    { heading: "Other matches", items: rest.map((entry) => entry.item), quiet: true },
    ...inText,
  ].filter((group) => group.items.length > 0);
};
