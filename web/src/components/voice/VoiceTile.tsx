import { VideoTrack, isTrackReference, useIsMuted, useIsSpeaking, useTrackRefContext } from "@livekit/components-react";
import { Track } from "livekit-client";
import { MicOff } from "lucide-react";
import type { CSSProperties } from "react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceTileProps {
  style: CSSProperties;
}

// One person in the call: their camera with a name chip over it, or their avatar with the name under it; a mute mark follows the name in both.
export const VoiceTile = ({ style }: VoiceTileProps) => {
  const trackRef = useTrackRefContext();
  const participant = trackRef.participant;
  const speaking = useIsSpeaking(participant);
  const micMuted = useIsMuted({ participant, source: Track.Source.Microphone });
  const cameraMuted = useIsMuted(trackRef);
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const name = `${participant.name || participant.identity}${participant.isLocal ? " (you)" : ""}`;
  const person = lookup(participant.identity);
  const video = isTrackReference(trackRef) && !cameraMuted;
  const avatarSize = Math.round(Math.min(64, Math.max(32, Number(style.height) * 0.32)));

  return (
    <div
      style={style}
      title={micMuted ? `${name}, muted` : name}
      // The speaking ring sits outside the tile on the well, so it reads the same over video as over a card.
      className={cn(
        "relative flex shrink-0 flex-col items-center justify-center gap-2 overflow-hidden rounded-md border border-border ring-2 ring-offset-2 ring-offset-surface-2 transition-shadow duration-150 ease-standard",
        video ? "bg-media" : "bg-card",
        speaking ? "ring-success" : "ring-transparent",
      )}
    >
      {video && <VideoTrack trackRef={trackRef} className="absolute inset-0 size-full object-cover" />}
      {video && (
        <span className="dark absolute bottom-2 left-2 flex h-5 max-w-[calc(100%-16px)] items-center gap-1 rounded-sm bg-media/80 px-1.5 text-xs font-medium text-white">
          <span className="truncate" dir="auto">{name}</span>
          {micMuted && <MicOff className="size-3 shrink-0 text-destructive" aria-label="Muted" />}
        </span>
      )}
      {!video && (
        <span style={{ "--avatar": `${avatarSize}px` } as CSSProperties}>
          <PersonAvatar login={person.login} src={person.avatar_url} label={name} className="size-(--avatar) text-base" />
        </span>
      )}
      {!video && (
        <span className="flex max-w-[calc(100%-16px)] items-center gap-1 text-xs text-muted-foreground">
          <span dir="auto" className="truncate">{name}</span>
          {micMuted && <MicOff className="size-3 shrink-0 text-destructive" aria-label="Muted" />}
        </span>
      )}
    </div>
  );
};
