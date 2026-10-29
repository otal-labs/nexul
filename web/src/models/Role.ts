import type { WelcomeStep } from "@nexul/client-core/welcome";

export interface Role {
  id: string;
  workspace_id: string;
  name: string;
  // One line on what holders do; empty for roles made before descriptions, which show their name alone.
  description: string;
  permissions: string[];
  // The Welcome wizard steps the role gives its holders; the Owner role's are fixed by the server.
  welcome_steps: WelcomeStep[];
  // Whether its holders see an action they can't take dimmed with the reason, instead of not at all.
  show_locked_actions: boolean;
  is_owner_role: boolean;
  created_at: string;
  updated_at: string;
}

// Mirrors roles.DescriptionMax on the server, which refuses a longer description.
export const ROLE_DESCRIPTION_MAX = 140;

// A duplicate's name: "Admin (copy)", then "Admin (copy 2)" and on while those are taken.
export const copyName = (name: string, taken: readonly string[]): string => {
  const base = `${name} (copy)`;
  if (!taken.includes(base)) return base;
  let n = 2;
  while (taken.includes(`${name} (copy ${n})`)) n++;
  return `${name} (copy ${n})`;
};

const DANGLING = new Set(["and", "or", "&", "of", "the", "for", "-", "–", "·"]);

// Greedy word wrap by character count, close enough for a short sans label; a word wider than a line breaks across lines.
const wrap = (words: string[], width: number): string[] =>
  words.flatMap((word) => word.match(new RegExp(`.{1,${width}}`, "g")) ?? [word]).reduce<string[]>((lines, word) => {
    const last = lines.at(-1);
    if (last !== undefined && last.length + 1 + word.length <= width) return [...lines.slice(0, -1), `${last} ${word}`];
    return [...lines, word];
  }, []);

// A role's name for a column head of `lines` lines of `width` characters: cut at a whole word, never a joining one, with its ellipsis.
export const headName = (name: string, width = 11, lines = 3): string => {
  const words = name.split(/\s+/);
  if (wrap(words, width).length <= lines) return name;
  if (words.length === 1) return `${name.slice(0, width * lines - 1)}…`;
  const fits = () => wrap(words, width).length <= lines && wrap(words, width).at(-1)!.length + 1 <= width;
  for (;;) {
    words.pop();
    while (words.length > 1 && DANGLING.has(words.at(-1)!.toLowerCase())) words.pop();
    words[words.length - 1] = words.at(-1)!.replace(/[,;:(]+$/, "");
    if (words.length === 1 || fits()) return `${words.join(" ")}…`;
  }
};
