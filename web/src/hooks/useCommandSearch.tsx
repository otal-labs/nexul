import { FileTextIcon, TicketIcon } from "lucide-react";
import { useNavigate } from "react-router";

import { slugify } from "@/components/doc/docHeadings";
import { useFetchDocsByProject } from "@/hooks/DocHooks";
import { useSearchMentions } from "@/hooks/MentionHooks";
import { useFetchTicketsByProject } from "@/hooks/TicketHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { CommandGroup, CommandItem } from "@/models/Command";
import type { MentionSearchResult } from "@/models/Mention";
import { ticketPath } from "@/models/Ticket";

const RECENT = 5;

// A text match opens where its words are: a ticket's note, or a doc's heading above them.
const hitPath = (hit: MentionSearchResult): string => {
  if (hit.type === "doc") return `/docs/${hit.id}${hit.match === "text" && hit.heading ? `#${slugify(hit.heading)}` : ""}`;
  const ticket = `/tickets/${hit.key ?? hit.id}`;
  return hit.match === "text" && hit.note_id ? `${ticket}/notes?note=${hit.note_id}` : ticket;
};

// Browsing, the current project's recently updated tickets and docs; typing, tickets and docs from the server's search,
// titles apart from text matches, in the server's order. Nothing is fetched while the palette is closed.
export const useCommandSearch = (query: string, open: boolean): { groups: CommandGroup[]; searching: boolean } => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { current } = useSidebarProject();
  const browsing = query.trim() === "";
  const projectId = open && browsing ? (current?.id ?? "") : "";
  const { data: tickets } = useFetchTicketsByProject(projectId || undefined);
  const { data: docs } = useFetchDocsByProject(projectId);
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
  const items = (type: MentionSearchResult["type"], text: boolean): CommandItem[] =>
    found
      .filter((hit) => hit.type === type && (hit.match === "text") === text)
      .map((hit) => ({
        id: `${type}:${hit.id}`,
        label: hit.title,
        icon: type === "ticket" ? TicketIcon : FileTextIcon,
        hint: hit.key ?? hit.status_label ?? "",
        ...(text && { detail: hit.heading ? `${hit.heading} · ${hit.snippet ?? ""}` : (hit.snippet ?? "") }),
        run: go(hitPath(hit)),
      }));

  return {
    groups: [
      { heading: "Tickets", items: items("ticket", false), searched: true },
      { heading: "Docs", items: items("doc", false), searched: true },
      { heading: "Found in tickets", items: items("ticket", true), searched: true },
      { heading: "Found in docs", items: items("doc", true), searched: true },
    ],
    searching: isFetching,
  };
};
