import { AgendaDueRow } from "@/components/calendar/AgendaDueRow";
import { AgendaMeetingRow } from "@/components/calendar/AgendaMeetingRow";
import type { AgendaItem } from "@/lib/agenda";

export const AgendaItemRow = ({ item }: { item: AgendaItem }) => (
  <>
    {item.kind === "due" && <AgendaDueRow ticket={item.ticket} />}
    {item.kind === "meeting" && <AgendaMeetingRow meeting={item.meeting} occurrence={item.occurrence} />}
  </>
);
