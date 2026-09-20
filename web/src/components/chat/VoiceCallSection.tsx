import { AlertTriangle, Loader2, Settings } from "lucide-react";
import { Link } from "react-router";
import type { Room } from "livekit-client";
import { useShallow } from "zustand/react/shallow";

import { ConnectedCall } from "@/components/chat/ConnectedCall";
import { Button } from "@/components/ui/button";
import { useChatAuthorLookup } from "@/hooks/ChatHooks";
import type { Conversation } from "@/models/Chat";
import { useVoiceCallStore, type VoiceCallStatus } from "@/stores/voiceCallStore";

interface VoiceCallSectionProps {
  workspaceId: string;
  conversation: Conversation;
  /** True once this voice channel is the call the user intends to join; false while just browsing text chat. */
  active: boolean;
}

type CallPhase = "joinPrompt" | "connecting" | "notConfigured" | "error" | "connected" | "hidden";

const deriveCallPhase = (active: boolean, status: VoiceCallStatus, room: Room | null): CallPhase => {
  if (!active) return "joinPrompt";
  if (status === "connecting" || status === "idle") return "connecting";
  if (status === "not_configured") return "notConfigured";
  if (status === "error") return "error";
  if (status === "connected" && room) return "connected";
  return "hidden";
};

// States read from voiceCallStore, which owns the connection; the call survives this panel closing.
export const VoiceCallSection = ({ workspaceId, conversation, active }: VoiceCallSectionProps) => {
  const { status, room, error, join, leave, retry } = useVoiceCallStore(
    useShallow((s) => ({
      status: s.status,
      room: s.room,
      error: s.error,
      join: s.join,
      leave: s.leave,
      retry: s.retry,
    })),
  );
  const resolveLogin = useChatAuthorLookup(workspaceId);
  const phase = deriveCallPhase(active, status, room);

  return (
    <>
      {phase === "joinPrompt" && (
        <div className="flex shrink-0 items-center justify-between gap-2 border-b border-border bg-muted/30 px-3 py-2">
          <span className="text-xs text-muted-foreground">Not connected to this voice channel</span>
          <Button size="sm" onClick={() => void join(conversation.id)}>
            Join call
          </Button>
        </div>
      )}
      {phase === "connecting" && (
        <div className="flex shrink-0 items-center justify-between gap-2 border-b border-border bg-muted/30 px-3 py-2">
          <span className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin motion-reduce:animate-none" aria-hidden />
            Connecting to {conversation.name}…
          </span>
          <Button size="sm" variant="ghost" onClick={leave}>
            Cancel
          </Button>
        </div>
      )}
      {/* Never a dead button — links straight to where an owner fixes it instead of a disabled "Join". */}
      {phase === "notConfigured" && (
        <div className="flex shrink-0 flex-col gap-2 border-b border-border bg-muted/30 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
          <span className="flex items-start gap-2 text-xs text-muted-foreground">
            <Settings className="mt-0.5 size-3.5 shrink-0" aria-hidden />
            Voice needs a LiveKit connector — ask a workspace owner to set one up in Settings.
          </span>
          <div className="flex shrink-0 items-center gap-2">
            <Button asChild size="sm" variant="outline">
              <Link to="/settings">Open settings</Link>
            </Button>
            <Button size="sm" variant="ghost" onClick={leave}>
              Dismiss
            </Button>
          </div>
        </div>
      )}
      {phase === "error" && (
        <div className="flex shrink-0 flex-col gap-2 border-b border-border bg-destructive/5 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
          <span className="flex items-start gap-2 text-xs text-destructive">
            <AlertTriangle className="mt-0.5 size-3.5 shrink-0" aria-hidden />
            {error ?? "Couldn't connect to the voice call."}
          </span>
          <div className="flex shrink-0 items-center gap-2">
            <Button size="sm" variant="outline" onClick={retry}>
              Retry
            </Button>
            <Button size="sm" variant="ghost" onClick={leave}>
              Dismiss
            </Button>
          </div>
        </div>
      )}
      {phase === "connected" && room && <ConnectedCall room={room} resolveLogin={resolveLogin} />}
    </>
  );
};
