import { Button } from "@/components/ui/button";
import { VoiceDockControls } from "@/components/voice/VoiceDockControls";
import { VoiceDockStatusRow } from "@/components/voice/VoiceDockStatusRow";
import type { VoiceCallStatus } from "@/stores/voiceCallStore";

interface VoiceDockExpandedProps {
  status: VoiceCallStatus;
  channelName: string;
  micEnabled: boolean;
  cameraEnabled: boolean;
  screenShareEnabled: boolean;
  onLeave: () => void;
  onRetry: () => void;
  onToggleMic: () => void;
  onToggleCamera: () => void;
  onToggleScreenShare: () => void;
}

export const VoiceDockExpanded = ({
  status,
  channelName,
  micEnabled,
  cameraEnabled,
  screenShareEnabled,
  onLeave,
  onRetry,
  onToggleMic,
  onToggleCamera,
  onToggleScreenShare,
}: VoiceDockExpandedProps) => (
  <>
    <VoiceDockStatusRow status={status} channelName={channelName} onLeave={onLeave} />
    {status === "connected" && (
      <VoiceDockControls
        micEnabled={micEnabled}
        cameraEnabled={cameraEnabled}
        screenShareEnabled={screenShareEnabled}
        onToggleMic={onToggleMic}
        onToggleCamera={onToggleCamera}
        onToggleScreenShare={onToggleScreenShare}
      />
    )}
    {status === "error" && (
      <Button size="sm" variant="outline" className="mt-1.5 h-7 w-full" onClick={onRetry}>
        Retry
      </Button>
    )}
  </>
);
