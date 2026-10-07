import type { ReactNode } from "react";
import { Text as Span, View } from "react-native";

import { openLink } from "@/components/chat/MessageMarkdown";
import { Text } from "@/components/ui/text";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { useClockStore } from "@/stores/clockStore";
import { parseDiscordMarkdown, type InlinePart, type MarkdownLine } from "@/lib/discordMarkdown";
import { formatDiscordTimestamp } from "@/lib/time";
import { cn } from "@/lib/utils";

const DiscordTime = ({ date, style }: { date: Date; style: string | undefined }) =>
  formatDiscordTimestamp(date, style, useClockStore((s) => s.now));

const LinkText = ({ url, children }: { url: string; children: ReactNode }) => {
  const canReadTickets = useAreaAccess()?.("tickets") ?? false;
  return (
    <Span role="link" onPress={() => openLink(url, canReadTickets)} className="underline">
      {children}
    </Span>
  );
};

const InlineParts = ({ parts }: { parts: InlinePart[] }) => parts.map((part, i) => <Inline key={i} part={part} />);

const Inline = ({ part }: { part: InlinePart }) => (
  <>
    {part.kind === "text" && part.text}
    {part.kind === "mention" && <Span className="font-semibold">{part.text}</Span>}
    {part.kind === "code" && <Span className="bg-surface-2 font-mono">{part.text}</Span>}
    {part.kind === "bold" && (
      <Span className="font-semibold">
        <InlineParts parts={part.parts} />
      </Span>
    )}
    {part.kind === "italic" && (
      <Span className="italic">
        <InlineParts parts={part.parts} />
      </Span>
    )}
    {part.kind === "link" && (
      <LinkText url={part.url}>
        <InlineParts parts={part.parts} />
      </LinkText>
    )}
    {part.kind === "url" && <LinkText url={part.url}>{part.url}</LinkText>}
    {part.kind === "time" && <DiscordTime date={part.date} style={part.style} />}
  </>
);

const Line = ({ line, textClassName }: { line: MarkdownLine; textClassName: string }) => (
  <>
    {line.kind === "blank" && <View className="h-2" />}
    {line.kind === "text" && (
      <Text className={textClassName}>
        <InlineParts parts={line.parts} />
      </Text>
    )}
    {line.kind === "item" && (
      <View className="flex-row gap-1.5 pl-1">
        <Text className={textClassName}>•</Text>
        <Text className={cn("min-w-0 flex-1", textClassName)}>
          <InlineParts parts={line.parts} />
        </Text>
      </View>
    )}
    {line.kind === "quote" && (
      <View className="border-l-2 border-border pl-2">
        <Text className={textClassName}>
          <InlineParts parts={line.parts} />
        </Text>
      </View>
    )}
  </>
);

interface DiscordMarkdownProps {
  text: string;
  mentionHandles?: string[];
  textClassName?: string;
}

// Bot content only; nested spans are bare Text so they inherit the line's size and color.
export const DiscordMarkdown = ({ text, mentionHandles, textClassName = "text-[15px] leading-[21px]" }: DiscordMarkdownProps) => (
  <View className="gap-0.5">
    {parseDiscordMarkdown(text, mentionHandles).map((line, i) => (
      <Line key={i} line={line} textClassName={textClassName} />
    ))}
  </View>
);
