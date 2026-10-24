import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

import type { CalendarRange, Meeting, Reply } from "@nexul/client-core/calendar";
import { conversationLabel } from "@nexul/client-core/chat";
import { httpUrl } from "@nexul/client-core/embed";
import { requires } from "@nexul/client-core/requires";

import { api } from "@/api/client";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useCurrentWorkspaceId } from "@/hooks/WorkspaceHooks";
import { defineQuery } from "@/lib/liveQuery";
import { agendaWindow } from "@/lib/agenda";
import type { Ticket } from "@/models/Ticket";

export const getMeetingsKey = "getMeetings";
export const getMeetingKey = "getMeeting";
export const getDueTicketsKey = "getDueTickets";

// A frame names the workspace, so only that workspace's agenda refetches; a reply covers every occurrence of the row.
const meetingsQuery = defineQuery({
  key: getMeetingsKey,
  fetch: (workspaceId: string | undefined, from: string, to: string) =>
    api.get<CalendarRange>(
      `/api/workspaces/${encodeURIComponent(workspaceId ?? "")}/meetings?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    ),
  refreshes: {
    "meeting.created": { key: (p) => p.meeting.workspace_id },
    "meeting.updated": { key: (p) => p.meeting.workspace_id },
    "meeting.cancelled": { key: (p) => p.meeting.workspace_id },
    "meeting.restored": { key: (p) => p.meeting.workspace_id },
    "meeting.replied": { key: (p) => p.workspace_id },
  },
});

// Today and the 29 days after it, in the phone's own zone; the window is stable for the day, so it is one cache entry.
export const useAgendaWindow = () => useMemo(() => agendaWindow(new Date()), []);

export const useFetchMeetings = () => {
  const workspaceId = useCurrentWorkspaceId();
  const { from, to } = useAgendaWindow();
  return useQuery({ ...meetingsQuery.options(workspaceId, from, to), enabled: !!workspaceId });
};

const meetingQuery = defineQuery({
  key: getMeetingKey,
  fetch: (id: string | undefined) => api.get<Meeting>(`/api/meetings/${encodeURIComponent(id ?? "")}`),
  refreshes: {
    "meeting.updated": { record: (p) => p.meeting.id },
    "meeting.cancelled": { record: (p) => p.meeting.id },
    "meeting.restored": { record: (p) => p.meeting.id },
    "meeting.replied": { record: (p) => p.meeting_id },
  },
});

export const useFetchMeeting = (id: string | undefined) => useQuery({ ...meetingQuery.options(id), enabled: !!id });

// The tickets due in the window, not in a project's list: it answers for the viewer across the whole workspace.
// A ticket frame carries no workspace, so every cached window refetches; only the open agenda is mounted to do it.
const dueTicketsQuery = defineQuery({
  key: getDueTicketsKey,
  fetch: (workspaceId: string | undefined, from: string, to: string) =>
    api.get<{ items: Ticket[] }>(
      `/api/tickets/due?workspace_id=${encodeURIComponent(workspaceId ?? "")}&from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    ),
  refreshes: {
    "ticket.created": "all",
    "ticket.updated": "all",
    "ticket.status_changed": "all",
    "ticket.developer_changed": "all",
    "ticket.tester_changed": "all",
    "ticket.due_date_changed": "all",
    "ticket.finished": "all",
    "ticket.deleted": "all",
  },
});

export const useFetchDueTickets = () => {
  const workspaceId = useCurrentWorkspaceId();
  const { fromDay, toDay } = useAgendaWindow();
  return useQuery({ ...dueTicketsQuery.options(workspaceId, fromDay, toDay), enabled: !!workspaceId });
};

// The reply comes back as the whole meeting, so the open screen takes it without another request.
export const useReplyToMeeting = () => {
  const client = useQueryClient();
  return useMutation({
    mutationFn: requires("none", ({ id, reply }: { id: string; reply: Exclude<Reply, "pending"> }) =>
      api.put<Meeting>(`/api/meetings/${id}/reply`, { reply })),
    onSuccess: (meeting) => client.setQueryData([getMeetingKey, meeting.id], meeting),
  });
};

export interface MeetingPlace {
  label: string;
  // Set when the place is a web link, which opens in the browser; a voice channel has no call screen on the phone.
  url?: string;
}

// The voice channel by name, else the link's host; undefined when it has neither. The channel list is only read for a voice meeting.
export const useMeetingPlace = (meeting: Meeting): MeetingPlace | undefined => {
  const { data: conversations } = useFetchConversations(meeting.voice_channel_id ? meeting.workspace_id : undefined);
  if (meeting.voice_channel_id) {
    const channel = conversations?.find((c) => c.id === meeting.voice_channel_id);
    return { label: channel ? conversationLabel(channel) : "Voice channel" };
  }
  const url = httpUrl(meeting.link);
  const host = url && /^https?:\/\/([^/?#]+)/i.exec(url)?.[1];
  if (url && host) return { label: host, url };
  return meeting.link ? { label: meeting.link } : undefined;
};
