import { LoadingPlaceholder, PlaceholderBar } from "@/components/LoadingPlaceholder";
import { cn } from "@/lib/utils";

const MESSAGES = [
  { name: "w-20", lines: ["w-3/4"] },
  { name: "w-28", lines: ["w-full", "w-1/2"] },
  { name: "w-16", lines: ["w-2/3"] },
  { name: "w-24", lines: ["w-5/6", "w-3/5"] },
];

// Sized like MessageRow: a 32px avatar, the author line, then the body lines.
const MessageSkeleton = ({ name, lines }: { name: string; lines: string[] }) => (
  <div className="flex gap-2 px-3 py-2">
    <PlaceholderBar className="size-8 shrink-0 rounded-full" />
    <div className="flex min-w-0 flex-1 flex-col gap-2 pt-0.5">
      <PlaceholderBar className={cn("h-3.5", name)} />
      {lines.map((width, line) => (
        <PlaceholderBar key={line} className={cn("h-3.5", width)} />
      ))}
    </div>
  </div>
);

interface ChatTranscriptSkeletonProps {
  /** Draws the conversation's header bar too, for while the conversation itself is still unknown. */
  header?: boolean;
}

// Messages settle at the bottom of a conversation, so the placeholder sits there too, on the transcript's column.
export const ChatTranscriptSkeleton = ({ header = false }: ChatTranscriptSkeletonProps) => (
  <LoadingPlaceholder label="Loading messages" className="flex min-h-0 flex-1 flex-col">
    {header && (
      <div className="h-12 shrink-0 border-b border-border">
        <div className="mx-auto flex h-full w-full max-w-3xl items-center gap-2 px-3">
          <PlaceholderBar className="size-4" />
          <PlaceholderBar className="h-4 w-32" />
        </div>
      </div>
    )}
    <div className="flex min-h-0 flex-1 flex-col justify-end overflow-hidden">
      <div className="mx-auto w-full max-w-[var(--chat-column,48rem)] pt-2 pb-3">
        {MESSAGES.map((message) => (
          <MessageSkeleton key={message.name} {...message} />
        ))}
      </div>
    </div>
  </LoadingPlaceholder>
);
