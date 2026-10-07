import { MessageLink } from "@/components/chat/MessageLink";
import { cn } from "@/lib/utils";
import { parseDiscordMarkdown, type InlinePart, type MarkdownLine } from "@/utils/DiscordMarkdownUtility";
import { formatDiscordTimestamp, formatFullTime } from "@/utils/TimeUtility";

const InlineParts = ({ parts }: { parts: InlinePart[] }) => parts.map((part, i) => <Inline key={i} part={part} />);

const Inline = ({ part }: { part: InlinePart }) => (
  <>
    {part.kind === "text" && part.text}
    {part.kind === "mention" && <strong className="font-semibold">{part.text}</strong>}
    {part.kind === "code" && <code className="rounded-sm bg-surface-2 px-1 py-px font-mono text-[0.85em]">{part.text}</code>}
    {part.kind === "bold" && (
      <strong className="font-semibold">
        <InlineParts parts={part.parts} />
      </strong>
    )}
    {part.kind === "italic" && (
      <em>
        <InlineParts parts={part.parts} />
      </em>
    )}
    {part.kind === "link" && (
      <a href={part.url} title={part.url} target="_blank" rel="noopener noreferrer" className="underline underline-offset-2">
        <InlineParts parts={part.parts} />
      </a>
    )}
    {part.kind === "url" && <MessageLink url={part.url} />}
    {part.kind === "time" && (
      <time dateTime={part.date.toISOString()} title={formatFullTime(part.date.toISOString())}>
        {formatDiscordTimestamp(part.date, part.style)}
      </time>
    )}
  </>
);

const Line = ({ line }: { line: MarkdownLine }) => (
  <>
    {line.kind === "blank" && <p aria-hidden className="h-2" />}
    {line.kind === "text" && (
      <p>
        <InlineParts parts={line.parts} />
      </p>
    )}
    {line.kind === "item" && (
      <p className="flex gap-1.5 pl-1">
        <span aria-hidden>•</span>
        <span className="min-w-0">
          <InlineParts parts={line.parts} />
        </span>
      </p>
    )}
    {line.kind === "quote" && (
      <p className="border-l-2 border-border pl-2">
        <InlineParts parts={line.parts} />
      </p>
    )}
  </>
);

interface DiscordMarkdownProps {
  text: string;
  mentionHandles?: string[];
  className?: string | undefined;
}

// Bot content only: people's and the Agent's messages keep their own renderers.
export const DiscordMarkdown = ({ text, mentionHandles, className }: DiscordMarkdownProps) => (
  <div className={cn("wrap-anywhere", className)}>
    {parseDiscordMarkdown(text, mentionHandles).map((line, i) => (
      <Line key={i} line={line} />
    ))}
  </div>
);
