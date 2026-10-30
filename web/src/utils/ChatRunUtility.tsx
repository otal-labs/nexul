import { isContinuation } from "@nexul/client-core/chat";
import { personLabel, type Person } from "@nexul/client-core/person";

import type { Message } from "@/models/Chat";
import { chatDayLabel } from "@/utils/ChatDayUtility";

// The name a run's header shows; none for a run drawn without one (your own bubbles, system lines).
const headerName = (message: Message, resolveAuthor: (authorId: string) => Person, currentUserId: string | undefined): string | undefined => {
  if (message.author_kind === "system") return undefined;
  if (message.author_kind === "bot") return message.author_name || "Bot";
  if (message.author_kind === "agent") return "Agent";
  if (message.author_id === currentUserId) return undefined;
  return personLabel(resolveAuthor(message.author_id));
};

export interface RunLabel {
  name: string;
  day: string;
}

// Each message's run as the pinned sender line names it ("Deploy Notifier", "Thu 8 Oct"), taken from the message that opened the run.
export const runLabels = (messages: Message[], resolveAuthor: (authorId: string) => Person, currentUserId: string | undefined): (RunLabel | undefined)[] => {
  const labels: (RunLabel | undefined)[] = [];
  messages.forEach((message, i) => {
    const name = headerName(message, resolveAuthor, currentUserId);
    labels.push(isContinuation(messages[i - 1], message) ? labels[i - 1] : name ? { name, day: chatDayLabel(message.created_at) } : undefined);
  });
  return labels;
};
