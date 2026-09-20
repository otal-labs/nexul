import { Mic, MicOff, ScreenShare, ScreenShareOff, Video, VideoOff } from "lucide-react";

import { Button } from "@/components/ui/button";

interface VoiceDockControlsProps {
  micEnabled: boolean;
  cameraEnabled: boolean;
  screenShareEnabled: boolean;
  onToggleMic: () => void;
  onToggleCamera: () => void;
  onToggleScreenShare: () => void;
}

export const VoiceDockControls = ({
  micEnabled,
  cameraEnabled,
  screenShareEnabled,
  onToggleMic,
  onToggleCamera,
  onToggleScreenShare,
}: VoiceDockControlsProps) => (
  <div className="mt-1.5 flex items-center gap-1">
    <Button
      variant="ghost"
      size="icon"
      className="size-7"
      aria-label={micEnabled ? "Mute microphone" : "Unmute microphone"}
      aria-pressed={!micEnabled}
      onClick={onToggleMic}
    >
      {micEnabled && <Mic className="size-3.5" aria-hidden />}
      {!micEnabled && <MicOff className="size-3.5 text-destructive" aria-hidden />}
    </Button>
    <Button
      variant="ghost"
      size="icon"
      className="size-7"
      aria-label={cameraEnabled ? "Turn off camera" : "Turn on camera"}
      aria-pressed={cameraEnabled}
      onClick={onToggleCamera}
    >
      {cameraEnabled && <Video className="size-3.5" aria-hidden />}
      {!cameraEnabled && <VideoOff className="size-3.5" aria-hidden />}
    </Button>
    <Button
      variant="ghost"
      size="icon"
      className="size-7"
      aria-label={screenShareEnabled ? "Stop screen share" : "Share screen"}
      aria-pressed={screenShareEnabled}
      onClick={onToggleScreenShare}
    >
      {screenShareEnabled && <ScreenShareOff className="size-3.5 text-primary" aria-hidden />}
      {!screenShareEnabled && <ScreenShare className="size-3.5" aria-hidden />}
    </Button>
  </div>
);
