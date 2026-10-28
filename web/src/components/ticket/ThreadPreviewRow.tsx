import { Bot } from "lucide-react";

import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import type { Message } from "@/models/Chat";
import { parseQuestionMessage } from "@/models/Question";
import { parseSecretMessage } from "@/models/Secret";

interface ThreadPreviewRowProps {
  message: Message;
  author: Person;
}

// A question or secret request is stored as fenced JSON; the line says it the way the thread does.
const lineOf = (body: string): string => {
  const question = parseQuestionMessage(body);
  if (question) return question.questions.map((q) => q.text).join(" ");
  const secret = parseSecretMessage(body);
  if (secret) return `Asked for ${secret.label}`;
  return body;
};

const nameOf = (message: Message, author: Person): string => {
  if (message.author_kind === "agent") return "Agent";
  if (message.author_kind === "bot") return message.author_name ?? author.login;
  return personLabel(author);
};

export const ThreadPreviewRow = ({ message, author }: ThreadPreviewRowProps) => {
  const isAgent = message.author_kind === "agent";
  const isSystem = message.author_kind === "system";
  return (
    <li className="flex min-w-0 items-center gap-2 py-1 text-sm">
      {isAgent && (
        <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-accent text-accent-foreground">
          <Bot className="size-3" aria-hidden />
        </span>
      )}
      {!isAgent && !isSystem && (
        <PersonAvatar login={author.login} src={message.author_avatar_url || author.avatar_url} className="size-5 text-[10px]" />
      )}
      {!isSystem && (
        <span dir="auto" className="shrink-0 font-medium text-foreground">
          {nameOf(message, author)}
        </span>
      )}
      <time dateTime={message.created_at} className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
        {formatUpdatedAgo(message.created_at)}
      </time>
      <span dir="auto" className="min-w-0 truncate text-muted-foreground">
        {lineOf(message.body)}
      </span>
    </li>
  );
};
