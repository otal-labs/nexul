import { bodyToMarkdown } from "@/utils/RichtextUtility";

const heading = /^(#{1,6})\s+(.*?)[\s#]*$/;

const headingOf = (line: string): { level: number; text: string } | null => {
  const match = heading.exec(line.trim());
  if (!match?.[1] || match[2] === undefined) return null;
  return { level: match[1].length, text: match[2].trim().toLowerCase() };
};

// The markdown under a ticket body's "Acceptance criteria" heading, up to the next heading of the same or a higher level.
export const acceptanceCriteria = (body: string): string => {
  const lines = bodyToMarkdown(body).split("\n");
  const start = lines.findIndex((line) => headingOf(line)?.text === "acceptance criteria");
  if (start < 0) return "";
  const level = headingOf(lines[start] ?? "")?.level ?? 1;
  const rest = lines.slice(start + 1);
  const end = rest.findIndex((line) => (headingOf(line)?.level ?? 7) <= level);
  return (end < 0 ? rest : rest.slice(0, end)).join("\n").trim();
};
