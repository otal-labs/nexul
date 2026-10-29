import { BoxIcon, BrainIcon, FileTextIcon, GitPullRequestIcon, HashIcon, TicketIcon, WorkflowIcon, type LucideIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { groupSearchHits, searchCut, type SearchHit, type SearchKind } from "@nexul/client-core/search";

import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { resolveStatus, useFetchProjectStatuses } from "@/hooks/StatusHooks";
import { useSearchWorkspace } from "@/hooks/SearchHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { CommandGroup, CommandItem } from "@/models/Command";
import { memoryPath, projectTokenById, type Project } from "@/models/Project";
import { ticketPath } from "@/models/Ticket";

const RECENT = 5;

const kindIcon: Record<SearchKind, LucideIcon> = {
  ticket: TicketIcon,
  doc: FileTextIcon,
  pull_request: GitPullRequestIcon,
  stack: BoxIcon,
  automation: WorkflowIcon,
  channel: HashIcon,
  memory: BrainIcon,
};

// The kind in words, for rows whose meta has no key to say it.
const kindWord: Partial<Record<SearchKind, string>> = { doc: "Doc", stack: "Stack", automation: "Automation", channel: "Channel", memory: "Memory" };

const hitPath = (hit: SearchHit, projects: Project[]): string => {
  if (hit.type === "ticket") return `/tickets/${hit.hint || hit.id}`;
  if (hit.type === "doc") return `/docs/${hit.id}`;
  if (hit.type === "pull_request") return `/tickets/${hit.ticket_id}`;
  if (hit.type === "stack") return `/stacks/${hit.id}`;
  if (hit.type === "automation") return `/automations/${hit.id}`;
  if (hit.type === "channel") return `/chat/${hit.id}`;
  return memoryPath(projectTokenById(projects, hit.project_id ?? ""), hit.id);
};

// Browsing, the current project's recently updated tickets and docs; typing, every kind the workspace search finds,
// a group per kind of title hits and then the hits found only in text. Nothing is fetched while the palette is closed.
interface CommandSearch {
  groups: CommandGroup[];
  searching: boolean;
  cut: boolean;
  // How many hits the search returned for the query; undefined until it has answered one of two characters or more.
  found: number | undefined;
}

export const useCommandSearch = (query: string, open: boolean): CommandSearch => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { projects, current } = useSidebarProject();
  const browsing = query.trim() === "";
  const projectId = open && browsing ? (current?.id ?? "") : "";
  const { data: tickets } = useFetchTicketsByProject(projectId || undefined);
  const { data: statuses } = useFetchProjectStatuses(projectId || undefined);
  const { data: docs } = useFetchDocsByProject(projectId);
  const { data: results, isFetching } = useSearchWorkspace(open ? query : "");
  const go = (path: string) => () => void navigate(wsPath(path));

  if (browsing) {
    const recentTickets = (tickets ?? []).map((ticket) => {
      const status = resolveStatus(statuses, ticket.status);
      return {
        at: ticket.updated_at,
        item: {
          id: `ticket:${ticket.id}`,
          label: ticket.title,
          icon: TicketIcon,
          hint: current ? `${current.prefix}-${ticket.number}` : "",
          ...(status && { status: { label: status.name, kind: status.kind } }),
          run: go(ticketPath(ticket, current?.prefix)),
        },
      };
    });
    const recentDocs = (docs ?? [])
      .filter((doc) => doc.can_open && !doc.archived)
      .map((doc) => ({
        at: doc.updated_at,
        item: { id: `doc:${doc.id}`, label: doc.title, icon: FileTextIcon, context: "Doc", updatedAt: doc.updated_at, run: go(`/docs/${doc.id}`) },
      }));
    const recent = [...recentTickets, ...recentDocs]
      .sort((a, b) => b.at.localeCompare(a.at))
      .slice(0, RECENT)
      .map((entry): CommandItem => entry.item);
    return { groups: [{ heading: "Recently updated", items: recent }], searching: false, cut: false, found: undefined };
  }

  const found = query.trim().length >= 2 ? results : undefined;
  // A doc or memory says its kind, the project when it is another one, and its age, as it does among the recent rows; a
  // ticket's or pull request's key already names its project.
  const elsewhere = (hit: SearchHit) =>
    (hit.type === "doc" || hit.type === "memory") && hit.project_id && hit.project_id !== current?.id
      ? projects?.find((project) => project.id === hit.project_id)?.name
      : undefined;
  const item = (hit: SearchHit): CommandItem => {
    const context = [kindWord[hit.type], elsewhere(hit)].filter(Boolean).join(" · ");
    const dated = (hit.type === "doc" || hit.type === "memory") && hit.updated_at;
    return {
      id: `${hit.type}:${hit.id}`,
      label: hit.title,
      icon: kindIcon[hit.type],
      ...(hit.hint && { hint: hit.hint }),
      ...(context && { context }),
      ...(hit.status_label && { status: { label: hit.status_label, kind: hit.status_kind ?? "" } }),
      ...(dated && { updatedAt: hit.updated_at }),
      run: go(hitPath(hit, projects ?? [])),
    };
  };
  const groups = groupSearchHits(found?.hits ?? []).map((group) => ({ heading: group.heading, items: group.hits.map(item) }));

  return { groups, searching: isFetching, cut: searchCut(found), found: found?.hits.length };
};
