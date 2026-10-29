import type { TicketCounts } from "@/models/Project";

// A project's ticket-counts read with nothing counted; a test merges in the maps it checks.
export const ticketCounts = (counts: Partial<TicketCounts> = {}): TicketCounts => ({
  tickets_by_status: {},
  tickets_by_category: {},
  tickets_by_type: {},
  tickets_by_label: {},
  ...counts,
});
