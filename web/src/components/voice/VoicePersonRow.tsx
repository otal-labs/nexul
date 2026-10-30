import { useIsMuted, useIsSpeaking } from "@livekit/components-react";
import { Track, type Participant } from "livekit-client";
import { MicOff, Video } from "lucide-react";

import { PersonAvatar } from "@/components/PersonAvatar";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import { cn } from "@/lib/utils";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoicePersonRowProps {
  participant: Participant;
}

// One person in the call list: avatar (ringed while they speak, as on the stage), name, a camera mark when it is on, the mute mark.
export const VoicePersonRow = ({ participant }: VoicePersonRowProps) => {
  const lookup = usePersonLookup(useWorkspaceStore((s) => s.selectedWorkspaceId));
  const speaking = useIsSpeaking(participant);
  const micMuted = useIsMuted({ participant, source: Track.Source.Microphone });
  const cameraOff = useIsMuted({ participant, source: Track.Source.Camera });
  const name = `${participant.name || participant.identity}${participant.isLocal ? " (you)" : ""}`;
  const person = lookup(participant.identity);
  return (
    <div title={name} className="relative flex h-8 min-w-0 shrink-0 items-center gap-2 rounded-sm px-2">
      <PersonAvatar
        login={person.login}
        src={person.avatar_url}
        label={name}
        className={cn("ring-2 ring-transparent ring-offset-1 ring-offset-popover transition-shadow duration-150 ease-standard", speaking && "ring-success")}
      />
      <span
        dir="auto"
        className={cn(
          "min-w-0 flex-1 truncate text-left text-[13px] text-foreground",
          micMuted && "pr-5.5",
          !cameraOff && "pr-11",
        )}
      >
        {name}
      </span>
      {/* Pinned columns, so every camera and every mute mark lines up; a name runs on under a column it doesn't use. */}
      {!cameraOff && <Video className="absolute right-7.5 size-3.5 text-muted-foreground" aria-label="Camera on" />}
      {micMuted && <MicOff className="absolute right-2 size-3.5 text-muted-foreground" aria-label="Muted" />}
    </div>
  );
};
