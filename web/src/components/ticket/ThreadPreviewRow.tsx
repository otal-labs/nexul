import { Bot } from "lucide-react";

import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { cn } from "@/lib/utils";
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
  const line = lineOf(message.body);
  return (
    <li className="col-span-full grid min-w-0 grid-cols-subgrid items-baseline py-1 text-sm">
      {isAgent && (
        <span className="flex size-5 shrink-0 items-center justify-center self-center rounded-full bg-accent text-accent-foreground">
          <Bot className="size-3" aria-hidden />
        </span>
      )}
      {!isAgent && !isSystem && (
        <PersonAvatar
          login={author.login}
          name={personLabel(author)}
          src={message.author_avatar_url || author.avatar_url}
          className="size-5 self-center text-[10px]"
        />
      )}
      {!isSystem && (
        <span dir="auto" className="max-w-40 justify-self-start truncate font-medium text-foreground" title={nameOf(message, author)}>
          {nameOf(message, author)}
        </span>
      )}
      <time
        dateTime={message.created_at}
        className={cn("font-mono text-xs text-muted-foreground tabular-nums", isSystem && "col-start-3")}
      >
        {formatUpdatedAgo(message.created_at)}
      </time>
      <span dir="auto" className="min-w-0 truncate text-muted-foreground" title={line}>
        {line}
      </span>
    </li>
  );
};
