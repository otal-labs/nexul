import { BrainIcon, FileTextIcon, TicketIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { useFetchMemories } from "@/hooks/MemoryHooks";
import { useSearchMentions } from "@/hooks/MentionHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { CommandGroup, CommandItem } from "@/models/Command";
import { memoryPath, projectTokenById } from "@/models/Project";
import { ticketPath } from "@/models/Ticket";

const RECENT = 5;

// Browsing, the current project's recently updated tickets and docs; typing, tickets and docs from the server's search
// and the workspace's memories by title. Nothing is fetched while the palette is closed.
export const useCommandSearch = (query: string, open: boolean): { groups: CommandGroup[]; searching: boolean } => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { projects, current } = useSidebarProject();
  const browsing = query.trim() === "";
  const projectId = open && browsing ? (current?.id ?? "") : "";
  const { data: tickets } = useFetchTicketsByProject(projectId || undefined);
  const { data: docs } = useFetchDocsByProject(projectId);
  const { data: memories } = useFetchMemories(open && !browsing ? workspaceId : "");
  const { data: hits, isFetching } = useSearchMentions(open ? query : "");
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
    return { groups: [{ heading: "Recently updated", items: recent }], searching: false };
  }

  const found = query.trim().length >= 2 ? (hits ?? []).filter((hit) => hit.can_open) : [];
  const ticketHits: CommandItem[] = found
    .filter((hit) => hit.type === "ticket")
    .map((hit) => ({ id: `ticket:${hit.id}`, label: hit.title, icon: TicketIcon, hint: hit.status_label ?? "", run: go(`/tickets/${hit.id}`) }));
  const docHits: CommandItem[] = found
    .filter((hit) => hit.type === "doc")
    .map((hit) => ({ id: `doc:${hit.id}`, label: hit.title, icon: FileTextIcon, run: go(`/docs/${hit.id}`) }));
  const memoryItems: CommandItem[] = (memories ?? []).map((memory) => ({
    id: `memory:${memory.id}`,
    label: memory.title,
    icon: BrainIcon,
    keywords: "memory",
    run: go(memoryPath(projectTokenById(projects ?? [], memory.project_id), memory.id)),
  }));

  return {
    groups: [
      { heading: "Tickets", items: ticketHits, searched: true },
      { heading: "Docs", items: docHits, searched: true },
      { heading: "Memories", items: memoryItems },
    ],
    searching: isFetching,
  };
};
