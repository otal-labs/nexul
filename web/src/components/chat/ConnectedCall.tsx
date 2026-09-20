import { RoomAudioRenderer, RoomContext } from "@livekit/components-react";
import { Mic, MicOff, PhoneOff, ScreenShare, ScreenShareOff, Video, VideoOff } from "lucide-react";
import type { Room } from "livekit-client";
import { useShallow } from "zustand/react/shallow";

import { VoiceCallControlButton } from "@/components/chat/VoiceCallControlButton";
import { VoiceCallStage } from "@/components/chat/VoiceCallStage";
import { Button } from "@/components/ui/button";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

interface ConnectedCallProps {
  room: Room;
  resolveLogin: (identity: string) => string;
}

// Built from shadcn Button, not the library's ControlBar, so controls read as Mono Console, not the default look.
export const ConnectedCall = ({ room, resolveLogin }: ConnectedCallProps) => {
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
    <RoomContext.Provider value={room}>
      {/* Remaps the library's --lk-* theme onto Mono Console tokens; bounded height leaves room for chat below. */}
      <div
        data-lk-theme="default"
        className="voice-call-stage flex h-[45vh] min-h-56 shrink-0 flex-col gap-2 border-b border-border bg-background p-2"
      >
        <RoomAudioRenderer />
        <VoiceCallStage resolveLogin={resolveLogin} />
        <div className="flex shrink-0 items-center justify-center gap-2 border-t border-border pt-2">
          <VoiceCallControlButton
            enabled={micEnabled}
            onClick={toggleMic}
            onLabel="Mute microphone"
            offLabel="Unmute microphone"
            OnIcon={Mic}
            OffIcon={MicOff}
          />
          <VoiceCallControlButton
            enabled={cameraEnabled}
            onClick={toggleCamera}
            onLabel="Turn off camera"
            offLabel="Turn on camera"
            OnIcon={Video}
            OffIcon={VideoOff}
          />
          <VoiceCallControlButton
            enabled={screenShareEnabled}
            onClick={toggleScreenShare}
            onLabel="Stop screen share"
            offLabel="Share screen"
            OnIcon={ScreenShare}
            OffIcon={ScreenShareOff}
          />
          <Button variant="destructive" size="icon" aria-label="Leave call" onClick={leave}>
            <PhoneOff className="size-4" aria-hidden />
          </Button>
        </div>
      </div>
    </RoomContext.Provider>
  );
};
