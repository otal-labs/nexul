import { useParticipants } from "@livekit/components-react";

import { VoiceCallButtons } from "@/components/voice/VoiceCallButtons";
import { VoiceCollapseButton } from "@/components/voice/VoiceCollapseButton";
import { VoiceLobbyRow } from "@/components/voice/VoiceLobbyRow";
import { VoiceRoomAvatars } from "@/components/voice/VoiceRoomAvatars";

// The call with the stage hidden: one row in the lobby row's slot, with who is in it and the same buttons.
export const VoiceCollapsedCall = () => {
  const participants = useParticipants();
  return (
    <VoiceLobbyRow
      lead={<VoiceRoomAvatars occupants={participants.map((p) => ({ identity: p.identity, name: p.name || p.identity }))} />}
      title={
        <>
          <span className="font-mono tabular-nums">{participants.length}</span> in the call
        </>
      }
      actions={
        <>
          <VoiceCallButtons />
          <VoiceCollapseButton />
        </>
      }
    />
  );
};
