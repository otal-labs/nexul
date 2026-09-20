import { PhoneOff } from "lucide-react";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { VoiceDockExpanded } from "@/components/voice/VoiceDockExpanded";
import { useFetchConversations } from "@/hooks/ChatHooks";
import { cn } from "@/lib/utils";
import { useVoiceCallStore } from "@/stores/voiceCallStore";
import { useWorkspaceStore } from "@/stores/workspaceStore";

interface VoiceDockProps {
  collapsed: boolean;
}

// A pure view over voiceCallStore (owns the LiveKit room); collapsed keeps the call alive, just shows leave.
export const VoiceDock = ({ collapsed }: VoiceDockProps) => {
  const { active, status, micEnabled, cameraEnabled, screenShareEnabled, leave, retry, toggleMic, toggleCamera, toggleScreenShare } =
    useVoiceCallStore(
      useShallow((s) => ({
        active: s.activeConversationId,
        status: s.status,
        micEnabled: s.micEnabled,
        cameraEnabled: s.cameraEnabled,
        screenShareEnabled: s.screenShareEnabled,
        leave: s.leave,
        retry: s.retry,
        toggleMic: s.toggleMic,
        toggleCamera: s.toggleCamera,
        toggleScreenShare: s.toggleScreenShare,
      })),
    );
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: conversations } = useFetchConversations(active ? workspaceId : undefined);

  if (!active) return null;
  const channelName = conversations?.find((c) => c.id === active)?.name ?? "voice";

  return (
    <div className={cn("border-t border-border", collapsed ? "flex justify-center p-2" : "px-2 py-2")}>
      {collapsed && (
        <Button variant="ghost" size="icon" aria-label="Leave voice" onClick={leave}>
          <PhoneOff className="size-4 text-destructive" aria-hidden />
        </Button>
      )}
      {!collapsed && (
        <VoiceDockExpanded
          status={status}
          channelName={channelName}
          micEnabled={micEnabled}
          cameraEnabled={cameraEnabled}
          screenShareEnabled={screenShareEnabled}
          onLeave={leave}
          onRetry={retry}
          onToggleMic={toggleMic}
          onToggleCamera={toggleCamera}
          onToggleScreenShare={toggleScreenShare}
        />
      )}
    </div>
  );
};
