import type { ComponentType } from "react";

export interface CommandItem {
  id: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  // Matched like the label, never shown: "palette colour" on a theme.
  keywords?: string;
  // Trails the label in mono: a ticket key, a project prefix.
  hint?: string;
  run: () => void;
}

export interface CommandGroup {
  heading: string;
  items: CommandItem[];
  // Already matched by the server (search results), so the typed query does not filter it again.
  searched?: boolean;
}

export const commandOptionId = (index: number) => `command-option-${index}`;

// Every word typed has to appear in the label, keywords or hint; a label that starts with the query ranks first.
const rank = (item: CommandItem, words: string[]): number => {
  const label = item.label.toLowerCase();
  const haystack = `${label} ${item.keywords ?? ""} ${item.hint ?? ""}`.toLowerCase();
  if (!words.every((word) => haystack.includes(word))) return -1;
  if (label.startsWith(words.join(" "))) return 2;
  if (label.split(/\s+/).some((part) => part.startsWith(words[0] ?? ""))) return 1;
  return 0;
};

export const filterCommandGroups = (groups: CommandGroup[], query: string): CommandGroup[] => {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  return groups
    .map((group) => {
      if (words.length === 0 || group.searched) return group;
      const items = group.items
        .map((item) => ({ item, score: rank(item, words) }))
        .filter((entry) => entry.score >= 0)
        .sort((a, b) => b.score - a.score)
        .map((entry) => entry.item);
      return { ...group, items };
    })
    .filter((group) => group.items.length > 0);
};
