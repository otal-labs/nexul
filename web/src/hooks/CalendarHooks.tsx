import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import type { CalendarRange, Meeting, Reply } from "@nexul/client-core/calendar";
import { requires } from "@nexul/client-core/requires";

import { api, errorMessage } from "@/api/client";
import { followEach, type LiveFollower } from "@/lib/live";
import type { MeetingBody, MeetingScope } from "@/models/Meeting";
import type { Ticket } from "@/models/Ticket";

export const getCalendarKey = "getCalendar";
export const getMeetingKey = "getMeeting";
export const getDueTicketsKey = "getDueTickets";

// The caller's meetings and every time each takes place between two instants; the server limits a range to 62 days.
export const useFetchCalendarRange = (workspaceId: string, from: string, to: string) =>
  useQuery({
    queryKey: [getCalendarKey, workspaceId, from, to],
    queryFn: async () => (await api.get<CalendarRange>(`/api/workspaces/${workspaceId}/meetings`, { params: { from, to } })).data,
    enabled: !!workspaceId,
  });

// The tickets the caller sees that are due between two days, both included, as YYYY-MM-DD.
export const useFetchDueTickets = (workspaceId: string, from: string, to: string) =>
  useQuery({
    queryKey: [getDueTicketsKey, workspaceId, from, to],
    queryFn: async () =>
      (await api.get<{ items: Ticket[] }>("/api/tickets/due", { params: { workspace_id: workspaceId, from, to } })).data.items,
    enabled: !!workspaceId,
  });

export const useFetchMeeting = (id: string) =>
  useQuery({
    queryKey: [getMeetingKey, id],
    queryFn: async () => (await api.get<Meeting>(`/api/meetings/${id}`)).data,
    enabled: !!id,
  });

const refreshCalendar = (client: QueryClient, workspaceId: string) =>
  client.invalidateQueries({ queryKey: [getCalendarKey, workspaceId] });

// The saved meeting is what the panel shows next, so it goes in the cache beside the refreshed ranges.
const meetingSaved = async (client: QueryClient, meeting: Meeting) => {
  client.setQueryData([getMeetingKey, meeting.id], meeting);
  await refreshCalendar(client, meeting.workspace_id);
};

export const useCreateMeeting = (workspaceId: string) => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("calendar:write", async (body: MeetingBody) =>
      (await api.post<Meeting>(`/api/workspaces/${workspaceId}/meetings`, body)).data),
    onSuccess: async (meeting) => {
      await meetingSaved(client, meeting);
      toast.success("Meeting created");
    },
  });
};

export interface MeetingChange extends Partial<MeetingBody> {
  // true cancels and false restores.
  cancelled?: boolean;
  // An occurrence of a repeating meeting, and which of its meetings the change covers.
  slot?: string;
  scope?: MeetingScope;
}

const changeToast = ({ cancelled }: MeetingChange) => {
  if (cancelled === true) return "Meeting cancelled";
  return cancelled === false ? "Meeting restored" : "Meeting saved";
};

// The change may land on another meeting than the one asked: a copy of one occurrence, or the series that carries on.
export const useUpdateMeeting = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("calendar:write", async ({ id, change }: { id: string; change: MeetingChange }) =>
      (await api.patch<Meeting>(`/api/meetings/${id}`, change)).data),
    onSuccess: async (meeting, { id, change }) => {
      await meetingSaved(client, meeting);
      if (meeting.id !== id) await client.invalidateQueries({ queryKey: [getMeetingKey, id], exact: true });
      toast.success(changeToast(change));
    },
  });
};

export const useReplyToMeeting = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", async ({ id, reply }: { id: string; reply: Exclude<Reply, "pending"> }) =>
      (await api.put<Meeting>(`/api/meetings/${id}/reply`, { reply })).data),
    onSuccess: (meeting) => meetingSaved(client, meeting),
    onError: (error) => toast.error(errorMessage(error)),
  });
};

interface MeetingPayload {
  meeting: { id: string; workspace_id: string };
}

interface ReplyPayload {
  meeting_id: string;
  workspace_id: string;
}

interface DueDatePayload {
  ticket: { due_on: string };
  previous_due_on: string;
}

const refreshMeeting = (client: QueryClient, workspaceId: string, id: string) =>
  Promise.all([refreshCalendar(client, workspaceId), client.invalidateQueries({ queryKey: [getMeetingKey, id], exact: true })]);

// A due-date query holds the days between its two ends, so only one reaching the old or the new day can have changed.
const reaches = (days: string[]) => ({ queryKey: [, , from, to] }: { queryKey: readonly unknown[] }) =>
  days.some((day) => !!day && String(from) <= day && day <= String(to));

// Only the calendar and the open meeting refresh: a frame names the meeting, and the ranges that show it can't be told apart.
export const calendarFollower: LiveFollower = {
  ...followEach(
    ["meeting.created", "meeting.updated", "meeting.cancelled", "meeting.restored"],
    ({ meeting }: MeetingPayload, { client }) => refreshMeeting(client, meeting.workspace_id, meeting.id),
  ),
  "meeting.replied": ({ meeting_id: id, workspace_id: workspaceId }: ReplyPayload, { client }) => refreshMeeting(client, workspaceId, id),
  // The frame names a ticket, not a workspace, so the due queries of every workspace are asked about the two days.
  "ticket.due_date_changed": ({ ticket, previous_due_on: previous }: DueDatePayload, { client }) =>
    client.invalidateQueries({ queryKey: [getDueTicketsKey], predicate: reaches([ticket.due_on, previous]) }),
};
