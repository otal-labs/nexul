import { Mic, MicOff, PhoneOff, ScreenShare, Video, VideoOff } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { VoiceControlButton } from "@/components/voice/VoiceControlButton";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

// Mic, camera, share and Leave: the same buttons on the stage's bar and on the collapsed row; muted is the one loud state.
export const VoiceCallButtons = () => {
  const { micEnabled, cameraEnabled, screenShareEnabled, leave, toggleMic, toggleCamera, toggleScreenShare } = useVoiceCallStore(
    useShallow((s) => ({
      micEnabled: s.micEnabled,
      cameraEnabled: s.cameraEnabled,
      screenShareEnabled: s.screenShareEnabled,
      leave: s.leave,
      toggleMic: s.toggleMic,
      toggleCamera: s.toggleCamera,
      toggleScreenShare: s.toggleScreenShare,
    })),
  );
  return (
    <div className="flex items-center gap-2">
      <VoiceControlButton
        label={micEnabled ? "Mute microphone" : "Unmute microphone"}
        icon={micEnabled ? Mic : MicOff}
        look={micEnabled ? "plain" : "alarm"}
        pressed={!micEnabled}
        onClick={toggleMic}
      />
      <VoiceControlButton
        label={cameraEnabled ? "Turn off camera" : "Turn on camera"}
        icon={cameraEnabled ? Video : VideoOff}
        look={cameraEnabled ? "checked" : "plain"}
        pressed={cameraEnabled}
        onClick={toggleCamera}
      />
      <VoiceControlButton
        label={screenShareEnabled ? "Stop sharing" : "Share screen"}
        icon={ScreenShare}
        look={screenShareEnabled ? "checked" : "plain"}
        pressed={screenShareEnabled}
        onClick={toggleScreenShare}
      />
      <span aria-hidden className="mx-1 h-6 w-px bg-border" />
      <VoiceControlButton label="Leave call" icon={PhoneOff} look="leave" onClick={leave} />
    </div>
  );
};
