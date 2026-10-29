import { personLabel } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePerson } from "@/hooks/PeopleHooks";
import { useFetchDocs } from "@/hooks/DocHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchTickets } from "@/hooks/TicketHooks";
import { cn } from "@/lib/utils";
import { dayLabel } from "@/utils/HistoryChangeUtility";

const PERSON_FIELDS = new Set(["developer", "tester"]);
const TICKET_FIELDS = new Set(["blocked_by", "found_in"]);
const DAY_FIELDS = new Set(["due_on", "finished_at"]);

// The person a login names, as the app shows people elsewhere: a small avatar, then their name.
const HistoryPersonValue = ({ login, avatar }: { login: string; avatar: boolean }) => {
  const person = usePerson(login);
  return (
    <>
      {avatar && <PersonAvatar login={person.login} src={person.avatar_url} name={personLabel(person)} label={personLabel(person)} className="size-3.5" />}
      <span className="min-w-0 truncate">{personLabel(person)}</span>
    </>
  );
};

// The ticket an id names, by the key the board shows; one the viewer can't list, or that was deleted, reads neutrally.
const HistoryTicketValue = ({ id }: { id: string }) => {
  const { data: tickets } = useFetchTickets();
  const { data: projects } = useFetchProjects();
  const ticket = tickets?.find((t) => t.id === id);
  const prefix = projects?.find((p) => p.id === ticket?.project_id)?.prefix;
  const key = ticket && prefix && `${prefix}-${ticket.number}`;
  return <span className={cn("min-w-0 truncate", key && "font-mono")}>{key || "Another ticket"}</span>;
};

const HistoryDocValue = ({ id }: { id: string }) => {
  const { data: docs } = useFetchDocs();
  return <span className="min-w-0 truncate">{docs?.find((d) => d.id === id)?.title || "Another doc"}</span>;
};

interface HistoryFieldValueProps {
  field: string;
  value: string;
  /** The side a change moved away from: muted, and without an avatar, so both names fit the rail. */
  muted?: boolean;
}

type ValueKind = "none" | "unknown" | "person" | "ticket" | "doc" | "day" | "text";

const kindOf = (field: string, value: string): ValueKind => {
  if (value === "") return "none";
  if (field === "found_in" && value === "unknown") return "unknown";
  if (PERSON_FIELDS.has(field)) return "person";
  if (TICKET_FIELDS.has(field)) return "ticket";
  if (field === "doc") return "doc";
  if (DAY_FIELDS.has(field)) return "day";
  return "text";
};

// One side of a field change in words the page already uses: names for logins, keys for ticket ids, titles for doc ids, "30 Oct 2026" for days.
export const HistoryFieldValue = ({ field, value, muted = false }: HistoryFieldValueProps) => {
  const kind = kindOf(field, value);
  return (
    <span className={cn("flex min-w-0 items-center gap-1", kind === "none" && "shrink-0", muted && "text-muted-foreground")}>
      {kind === "none" && "None"}
      {kind === "unknown" && <span className="min-w-0 truncate">Unknown</span>}
      {kind === "person" && <HistoryPersonValue login={value} avatar={!muted} />}
      {kind === "ticket" && <HistoryTicketValue id={value} />}
      {kind === "doc" && <HistoryDocValue id={value} />}
      {kind === "day" && <span className="min-w-0 truncate">{dayLabel(value)}</span>}
      {kind === "text" && <span className="min-w-0 truncate">{value}</span>}
    </span>
  );
};
