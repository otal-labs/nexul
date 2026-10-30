import { Maximize2, Minimize2 } from "lucide-react";

import { VoiceControlButton } from "@/components/voice/VoiceControlButton";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

export const VoiceCollapseButton = () => {
  const collapsed = useVoiceCallStore((s) => s.stageCollapsed);
  const toggle = useVoiceCallStore((s) => s.toggleStageCollapsed);
  return <VoiceControlButton label={collapsed ? "Show call" : "Hide call"} icon={collapsed ? Maximize2 : Minimize2} look="plain" onClick={toggle} />;
};
