import { useQueryClient, type QueryClient } from "@tanstack/react-query";

import { conversationLabel, type Conversation } from "@nexul/client-core/chat";
import { unknownPerson, type Person } from "@nexul/client-core/person";

import { getMeKey } from "@/hooks/AuthHooks";
import { getChatConversationsKey } from "@/hooks/ChatHooks";
import { getDocKey, getDocsKey } from "@/hooks/DocHooks";
import { getWorkspacePeopleKey } from "@/hooks/PeopleHooks";
import { getTicketKey, getTicketsKey } from "@/hooks/TicketCache";
import { channelMention } from "@/models/Chat";
import { parseTicketKey } from "@/models/Ticket";
import type { MeResponse } from "@/models/User";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { InstanceLink } from "@/utils/MessageTextUtility";

const SECTION_LABELS: Record<string, string> = {
  automations: "Automation",
  board: "Board",
  chat: "Chat",
  configuration: "Configuration",
  docs: "Doc",
  inbox: "Inbox",
  memories: "Memories",
  projects: "Project",
  runners: "Runners",
  settings: "Settings",
  stacks: "Stack",
  tickets: "Ticket",
  topology: "Topology",
  wizard: "Wizard",
};

const findCached = <T extends { id: string }>(client: QueryClient, listKey: string, oneKey: string | null, id: string): T | undefined =>
  (oneKey ? client.getQueryData<T>([oneKey, id]) : undefined) ??
  client
    .getQueriesData<T[]>({ queryKey: [listKey] })
    .flatMap(([, list]) => (Array.isArray(list) ? list : []))
    .find((item) => item.id === id);

const conversationName = (client: QueryClient, c: Conversation): string => {
  if (c.kind === "channel" || c.kind === "voice_channel") return channelMention(c);
  const people = client.getQueryData<Person[]>([getWorkspacePeopleKey, c.workspace_id]) ?? [];
  const resolvePerson = (id: string) => people.find((p) => p.user_id === id) ?? unknownPerson(id);
  return conversationLabel(c, { currentUserId: client.getQueryData<MeResponse>([getMeKey])?.user.id, resolvePerson });
};

const cachedLabel = (client: QueryClient, section: string, id: string): string | undefined => {
  if (section === "chat") {
    const c = findCached<Conversation>(client, getChatConversationsKey, null, id);
    return c && conversationName(client, c);
  }
  if (section === "docs") return findCached<{ id: string; title: string }>(client, getDocsKey, getDocKey, id)?.title;
  if (section === "tickets") return findCached<{ id: string; title: string }>(client, getTicketsKey, getTicketKey, id)?.title;
  return undefined;
};

// Reads only what is already cached, never fetches: a miss falls back to the page's kind, or the path for an unknown page.
export const useInstanceLinkLabel = (link: InstanceLink): string => {
  const client = useQueryClient();
  const currentSlug = useWorkspaceStore((s) => s.selectedWorkspaceSlug);
  const { section, rest, workspace } = link;
  const fallback = SECTION_LABELS[section ?? ""] ?? link.path;
  const id = section === "docs" ? rest.at(-1) : rest[0];
  if (!section || !id) return fallback;
  if (section === "tickets" && parseTicketKey(id)) return id;
  if (workspace !== currentSlug) return fallback;
  return cachedLabel(client, section, id) || fallback;
};
