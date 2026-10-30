import { VoiceCallButtons } from "@/components/voice/VoiceCallButtons";
import { VoiceCallTimer } from "@/components/voice/VoiceCallTimer";
import { VoiceCollapseButton } from "@/components/voice/VoiceCollapseButton";
import { VoiceViewButton } from "@/components/voice/VoiceViewButton";

// The call's controls, centred inside the stage well under a hairline: the call's clock at the left end, the view and hide buttons at the right.
export const VoiceCallControls = () => (
  <div className="grid h-12 shrink-0 grid-cols-[1fr_auto_1fr] items-center gap-2 border-t border-border pt-3">
    <VoiceCallTimer />
    <div className="col-start-2">
      <VoiceCallButtons />
    </div>
    <div className="flex items-center justify-end gap-2">
      <VoiceViewButton />
      <VoiceCollapseButton />
    </div>
  </div>
);
