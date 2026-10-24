import { ExternalLinkIcon, Volume2Icon } from "lucide-react";
import { Link } from "react-router";

import type { Meeting } from "@nexul/client-core/calendar";

import { Fact } from "@/components/Fact";
import { Button } from "@/components/ui/button";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { channelMention } from "@/models/Chat";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

// A voice channel joins the call and opens the channel, as its sidebar row does; a link opens in a new tab.
export const MeetingWhereFact = ({ meeting }: { meeting: Meeting }) => {
  const wsPath = useWorkspacePath();
  const joinCall = useVoiceCallStore((s) => s.join);
  const { data: conversations } = useFetchConversations(meeting.workspace_id);
  const channel = conversations?.find((c) => c.id === meeting.voice_channel_id);
  const name = channel?.name ?? "Voice channel";
  return (
    <>
      {meeting.voice_channel_id && (
        <Fact label="Where">
          <Button asChild variant="outline" size="sm">
            <Link to={wsPath(`/chat/${meeting.voice_channel_id}`)} onClick={() => void joinCall(meeting.voice_channel_id, channel ? channelMention(channel) : name)}>
              <Volume2Icon aria-hidden />
              Join {name}
            </Link>
          </Button>
        </Fact>
      )}
      {meeting.link && (
        <Fact label="Where">
          <Button asChild variant="outline" size="sm">
            <a href={meeting.link} target="_blank" rel="noopener noreferrer">
              <ExternalLinkIcon aria-hidden />
              Open link
            </a>
          </Button>
        </Fact>
      )}
    </>
  );
};
