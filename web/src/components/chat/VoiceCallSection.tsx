import { AlertTriangle, Clock, Loader2, Unplug, UserRound } from "lucide-react";
import { useEffect } from "react";
import { Link } from "react-router";
import type { Room } from "livekit-client";
import { useShallow } from "zustand/react/shallow";

import type { Conversation } from "@nexul/client-core/chat";

import { ConnectedCall } from "@/components/chat/ConnectedCall";
import { Button } from "@/components/ui/button";
import { VoiceDismissButton } from "@/components/voice/VoiceDismissButton";
import { VoiceJoinMutedToggle } from "@/components/voice/VoiceJoinMutedToggle";
import { VoiceLobbyIcon } from "@/components/voice/VoiceLobbyIcon";
import { VoiceLobbyRow } from "@/components/voice/VoiceLobbyRow";
import { VoiceRoomAvatars } from "@/components/voice/VoiceRoomAvatars";
import { useCanOpenSection } from "@/hooks/AccessHooks";
import { useVoiceOccupancy } from "@/hooks/VoiceHooks";
import { channelMention } from "@/models/Chat";
import { roomNames } from "@/models/Voice";
import { useVoiceCallStore, type VoiceCallStatus } from "@/stores/voiceCallStore";

interface VoiceCallSectionProps {
  conversation: Conversation;
  /** True once this voice channel is the call the user intends to join; false while just browsing text chat. */
  active: boolean;
}

type CallPhase = "leftAlone" | "joinPrompt" | "connecting" | "notConfigured" | "error" | "connected" | "hidden";

const deriveCallPhase = (active: boolean, leftAlone: boolean, status: VoiceCallStatus, room: Room | null): CallPhase => {
  if (!active && leftAlone) return "leftAlone";
  if (!active) return "joinPrompt";
  if (status === "connecting" || status === "idle") return "connecting";
  if (status === "not_configured") return "notConfigured";
  if (status === "error") return "error";
  if (status === "connected" && room) return "connected";
  return "hidden";
};

const callCount = (n: number) => (
  <>
    <span className="font-mono tabular-nums">{n}</span> in the call
  </>
);

// States read from voiceCallStore, which owns the connection; the call survives this panel closing.
export const VoiceCallSection = ({ conversation, active }: VoiceCallSectionProps) => {
  const canConfigure = useCanOpenSection("connectors");
  const occupants = useVoiceOccupancy()[conversation.id] ?? [];
  const { leftAlone, status, room, error, otherCall, join, leave, retry, clearLeftAlone } = useVoiceCallStore(
    useShallow((s) => ({
      otherCall: s.activeConversationId && s.activeConversationId !== conversation.id ? s.activeChannelName : undefined,
      leftAlone: s.leftAlone?.conversationId === conversation.id ? s.leftAlone.reason : undefined,
      status: s.status,
      room: s.room,
      error: s.error,
      join: s.join,
      leave: s.leave,
      retry: s.retry,
      clearLeftAlone: s.clearLeftAlone,
    })),
  );
  const phase = deriveCallPhase(active, leftAlone !== undefined, status, room);
  const joinThis = () => void join(conversation.id, channelMention(conversation));
  const empty = occupants.length === 0;
  // The reason stays only while this channel stays open; moving to another one drops it.
  useEffect(() => () => clearLeftAlone(conversation.id), [clearLeftAlone, conversation.id]);

  return (
    <>
      {phase === "leftAlone" && (
        <VoiceLobbyRow
          lead={<VoiceLobbyIcon icon={Clock} />}
          title="You left after 5 minutes alone"
          detail={!empty && callCount(occupants.length)}
          actions={
            <>
              <VoiceJoinMutedToggle />
              <Button size="sm" onClick={joinThis}>Rejoin</Button>
              <VoiceDismissButton onClick={() => clearLeftAlone(conversation.id)} />
            </>
          }
        />
      )}
      {phase === "joinPrompt" && empty && (
        <VoiceLobbyRow
          lead={
            <span className="flex size-7 shrink-0 items-center justify-center rounded-full border border-dashed border-border">
              <UserRound className="size-3.5 text-muted-foreground" aria-hidden />
            </span>
          }
          title="Nobody's here"
          detail={otherCall ? `You're in ${otherCall}` : "Start the call and others can join"}
          actions={
            <>
              <VoiceJoinMutedToggle />
              <Button size="sm" onClick={joinThis}>{otherCall ? "Switch call" : "Start call"}</Button>
            </>
          }
        />
      )}
      {phase === "joinPrompt" && !empty && (
        <VoiceLobbyRow
          lead={<VoiceRoomAvatars occupants={occupants} />}
          title={roomNames(occupants.map((o) => o.name))}
          detail={
            <>
              {callCount(occupants.length)}
              {otherCall && ` · You're in ${otherCall}`}
            </>
          }
          actions={
            <>
              <VoiceJoinMutedToggle />
              <Button size="sm" onClick={joinThis}>{otherCall ? "Switch call" : "Join"}</Button>
            </>
          }
        />
      )}
      {phase === "connecting" && (
        <VoiceLobbyRow
          lead={<VoiceLobbyIcon icon={Loader2} className="size-4 animate-spin text-muted-foreground motion-reduce:animate-none" />}
          title={`Connecting to ${conversation.name ?? "the call"}…`}
          actions={<Button size="sm" variant="ghost" onClick={leave}>Cancel</Button>}
        />
      )}
      {/* Never a dead button: whoever can fix it gets the link, everyone else is told who can. */}
      {phase === "notConfigured" && (
        <VoiceLobbyRow
          lead={<VoiceLobbyIcon icon={Unplug} />}
          title="Voice isn't set up"
          detail={!canConfigure && "Ask someone who manages connectors to add LiveKit"}
          actions={
            <>
              {canConfigure && (
                <Button asChild size="sm" variant="outline">
                  <Link to="/settings/connectors">Open connectors</Link>
                </Button>
              )}
              <VoiceDismissButton onClick={leave} />
            </>
          }
        />
      )}
      {phase === "error" && (
        <VoiceLobbyRow
          lead={<VoiceLobbyIcon icon={AlertTriangle} className="size-4 text-destructive" />}
          title="Couldn't connect"
          detail={error && <span title={error} className="font-mono">{error}</span>}
          actions={
            <>
              <Button size="sm" onClick={retry}>
                Retry
              </Button>
              <VoiceDismissButton onClick={leave} />
            </>
          }
        />
      )}
      {phase === "connected" && room && <ConnectedCall room={room} />}
    </>
  );
};
