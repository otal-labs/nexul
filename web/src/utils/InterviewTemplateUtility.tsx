// Mirrors internal/memories/questions.go for display; the server's parse stays the one that saves or refuses.
export interface TemplateQuestionView {
  title: string;
  hint: string;
  multiSelect: boolean;
  options: string[];
}

const isHeading = (line: string) => {
  const trimmed = line.trim();
  return trimmed === "##" || trimmed.startsWith("## ");
};

// Splits the body at each ## heading; joining the parts with "\n" gives the body back byte for byte.
export const splitTemplate = (body: string): string[] =>
  body.split("\n").reduce<string[][]>((parts, line) => {
    if (isHeading(line) || parts.length === 0) return [...parts, [line]];
    parts[parts.length - 1]!.push(line);
    return parts;
  }, []).map((lines) => lines.join("\n"));

export const joinTemplate = (parts: string[]): string => parts.join("\n");

export const readQuestion = (part: string): TemplateQuestionView | null => {
  const lines = part.split("\n").map((line) => line.trim());
  if (!isHeading(lines[0] ?? "")) return null;
  const hint: string[] = [];
  const options: string[] = [];
  let multiSelect = false;
  let bulleted = false;
  for (const line of lines.slice(1)) {
    if (line === "") continue;
    if (line === "-" || line.startsWith("- ")) {
      const text = line.slice(1).trim();
      const multi = text === "[ ]" || text.startsWith("[ ] ");
      if (!bulleted) multiSelect = multi;
      bulleted = true;
      const label = (multi ? text.slice(3) : text).split(": ")[0]!.trim();
      if (label !== "") options.push(label);
      continue;
    }
    if (!bulleted) hint.push(line);
  }
  return { title: lines[0]!.slice(2).trim(), hint: hint.join(" "), multiSelect, options };
};
