import { ScreenShare } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useVoiceCallStore } from "@/stores/voiceCallStore";

// Your own shared screen is not played back to you: you already see it, and a mirror of it would loop.
export const VoiceOwnShare = () => {
  const stop = useVoiceCallStore((s) => s.toggleScreenShare);
  return (
    <div data-own-share="" className="flex flex-col items-center justify-center gap-3 text-center">
      <ScreenShare className="size-6 text-white/60" aria-hidden />
      <p className="text-sm text-white">You're sharing your screen</p>
      <Button size="sm" variant="outline" onClick={stop} className="border-white/20 bg-white/10 text-white shadow-none hover:bg-white/20 hover:text-white">
        Stop sharing
      </Button>
    </div>
  );
};
