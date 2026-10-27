import { BoxIcon, BrainIcon, FileTextIcon, GitPullRequestIcon, HashIcon, TicketIcon, WorkflowIcon, type LucideIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { groupSearchHits, searchCut, type SearchHit, type SearchKind } from "@nexul/client-core/search";

import { useFetchDocsByProject } from "@/hooks/DocHooks";
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
export const useCommandSearch = (query: string, open: boolean): { groups: CommandGroup[]; searching: boolean; cut: boolean } => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { projects, current } = useSidebarProject();
  const browsing = query.trim() === "";
  const projectId = open && browsing ? (current?.id ?? "") : "";
  const { data: tickets } = useFetchTicketsByProject(projectId || undefined);
  const { data: docs } = useFetchDocsByProject(projectId);
  const { data: results, isFetching } = useSearchWorkspace(open ? query : "");
  const go = (path: string) => () => void navigate(wsPath(path));

  if (browsing) {
    const recentTickets = (tickets ?? []).map((ticket) => ({
      at: ticket.updated_at,
      item: {
        id: `ticket:${ticket.id}`,
        label: ticket.title,
        icon: TicketIcon,
        hint: current ? `${current.prefix}-${ticket.number}` : "",
        run: go(ticketPath(ticket, current?.prefix)),
      },
    }));
    const recentDocs = (docs ?? [])
      .filter((doc) => doc.can_open && !doc.archived)
      .map((doc) => ({ at: doc.updated_at, item: { id: `doc:${doc.id}`, label: doc.title, icon: FileTextIcon, run: go(`/docs/${doc.id}`) } }));
    const recent = [...recentTickets, ...recentDocs]
      .sort((a, b) => b.at.localeCompare(a.at))
      .slice(0, RECENT)
      .map((entry): CommandItem => entry.item);
    return { groups: [{ heading: "Recently updated", items: recent }], searching: false, cut: false };
  }

  const found = query.trim().length >= 2 ? results : undefined;
  const item = (hit: SearchHit): CommandItem => ({
    id: `${hit.type}:${hit.id}`,
    label: hit.title,
    icon: kindIcon[hit.type],
    hint: hit.hint ?? hit.status_label ?? "",
    run: go(hitPath(hit, projects ?? [])),
  });
  const groups = groupSearchHits(found?.hits ?? []).map((group) => ({ heading: group.heading, items: group.hits.map(item), searched: true }));

  return { groups, searching: isFetching, cut: searchCut(found) };
};
