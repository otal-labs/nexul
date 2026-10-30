import { Mic, MicOff, ScreenShare, Video, VideoOff } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

// The stage's looks at dock size: a muted mic in colour, a camera or share that is on pressed in ink.
const ALARM = "bg-destructive/12 text-destructive hover:bg-destructive/20 hover:text-destructive";
const PRESSED = "bg-foreground/10 text-foreground hover:bg-foreground/15";

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
      className={cn("size-7", !micEnabled && ALARM)}
      aria-label={micEnabled ? "Mute microphone" : "Unmute microphone"}
      aria-pressed={!micEnabled}
      onClick={onToggleMic}
    >
      {micEnabled && <Mic className="size-3.5" aria-hidden />}
      {!micEnabled && <MicOff className="size-3.5" aria-hidden />}
    </Button>
    <Button
      variant="ghost"
      size="icon"
      className={cn("size-7", cameraEnabled && PRESSED)}
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
      className={cn("size-7", screenShareEnabled && PRESSED)}
      aria-label={screenShareEnabled ? "Stop screen share" : "Share screen"}
      aria-pressed={screenShareEnabled}
      onClick={onToggleScreenShare}
    >
      <ScreenShare className="size-3.5" aria-hidden />
    </Button>
  </div>
);
