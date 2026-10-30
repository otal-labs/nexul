import { LayoutGrid, LayoutPanelTop } from "lucide-react";

import { VoiceControlButton } from "@/components/voice/VoiceControlButton";
import { useVoiceStageViewStore } from "@/stores/voiceStageViewStore";

// Names the view it switches to; pressed while the speaker view is on.
export const VoiceViewButton = () => {
  const view = useVoiceStageViewStore((s) => s.view);
  const toggle = useVoiceStageViewStore((s) => s.toggle);
  const speaker = view === "speaker";
  return (
    <VoiceControlButton
      label={speaker ? "Grid view" : "Speaker view"}
      icon={speaker ? LayoutGrid : LayoutPanelTop}
      look={speaker ? "checked" : "plain"}
      pressed={speaker}
      onClick={toggle}
    />
  );
};
