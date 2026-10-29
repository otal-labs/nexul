import { groupReplies, type Meeting } from "@nexul/client-core/calendar";

import { useFetchConversations } from "@/hooks/ChatHooks";

// Where a meeting happens and how many have said they're coming, for the week's wide rows.
export const MeetingMeta = ({ meeting }: { meeting: Meeting }) => {
  const { data: conversations } = useFetchConversations(meeting.workspace_id);
  const channel = meeting.voice_channel_id ? (conversations?.find((c) => c.id === meeting.voice_channel_id)?.name ?? "Voice channel") : "";
  const coming = groupReplies(meeting).accepted.length;
  const parts = [channel, meeting.link && "Link", coming > 0 && `${coming} coming`].filter(Boolean);
  return <span className="relative top-px hidden max-w-64 truncate text-right text-xs text-muted-foreground @min-[68.75rem]:block">{parts.join(" · ")}</span>;
};
