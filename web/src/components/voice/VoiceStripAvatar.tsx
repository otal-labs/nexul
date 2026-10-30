import { useIsMuted, useIsSpeaking } from "@livekit/components-react";
import { Track, type Participant } from "livekit-client";
import { MicOff } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceStripAvatarProps {
  participant: Participant;
}

// A person in the narrow share strip: the avatar carries the speaking ring and a mute badge; the name is its tooltip.
export const VoiceStripAvatar = ({ participant }: VoiceStripAvatarProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const speaking = useIsSpeaking(participant);
  const micMuted = useIsMuted({ participant, source: Track.Source.Microphone });
  const name = participant.name || participant.identity;
  const person = lookup(participant.identity);
  return (
    <span className="relative shrink-0" title={micMuted ? `${name}, muted` : name}>
      <PersonAvatar
        login={person.login}
        name={name}
        src={person.avatar_url}
        label={name}
        className={cn("size-6 ring-2 ring-surface-2 transition-shadow duration-150 ease-standard", speaking && "ring-success")}
      />
      {micMuted && (
        <span className="absolute -right-1 -bottom-1 flex size-3.5 items-center justify-center rounded-full bg-surface-2">
          <MicOff className="size-2.5 text-muted-foreground" aria-hidden />
        </span>
      )}
    </span>
  );
};
