import type { Mic } from "lucide-react";

import { Button } from "@/components/ui/button";

interface VoiceCallControlButtonProps {
  /** Whether the track (mic/camera/screen share) is currently on. */
  enabled: boolean;
  onClick: () => void;
  onLabel: string;
  offLabel: string;
  OnIcon: typeof Mic;
  OffIcon: typeof Mic;
}

// The "off" state gets the emphasized (secondary) fill so a muted mic/camera reads as notable at a glance.
export const VoiceCallControlButton = ({ enabled, onClick, onLabel, offLabel, OnIcon, OffIcon }: VoiceCallControlButtonProps) => (
  <Button
    variant={enabled ? "outline" : "secondary"}
    size="icon"
    aria-label={enabled ? onLabel : offLabel}
    aria-pressed={!enabled}
    onClick={onClick}
  >
    {enabled && <OnIcon className="size-4 text-muted-foreground" aria-hidden />}
    {!enabled && <OffIcon className="size-4" aria-hidden />}
  </Button>
);
