import { Mic, MicOff } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { VoiceControlButton } from "@/components/voice/VoiceControlButton";
import { useJoinMutedStore } from "@/stores/joinMutedStore";

// Beside the lobby's join button: whether the mic starts off, remembered across reloads.
export const VoiceJoinMutedToggle = () => {
  const { joinMuted, toggleJoinMuted } = useJoinMutedStore(
    useShallow((s) => ({ joinMuted: s.joinMuted, toggleJoinMuted: s.toggleJoinMuted })),
  );
  return (
    <VoiceControlButton
      compact
      label={joinMuted ? "Join with mic on" : "Join muted"}
      icon={joinMuted ? MicOff : Mic}
      look={joinMuted ? "alarm" : "plain"}
      pressed={joinMuted}
      onClick={toggleJoinMuted}
    />
  );
};
