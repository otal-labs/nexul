import type { LucideIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

type Look = "plain" | "alarm" | "checked" | "leave";

interface VoiceControlButtonProps {
  label: string;
  icon: LucideIcon;
  look: Look;
  /** Omitted for an action, not a toggle. */
  pressed?: boolean;
  onClick: () => void;
}

// Only a muted mic is in colour; a camera or a share that is on reads as pressed, in ink.
const LOOKS: Record<Look, string> = {
  plain: "",
  alarm: "border-transparent bg-destructive/12 text-destructive shadow-none hover:bg-destructive/20 hover:text-destructive",
  checked: "border-foreground/25 bg-foreground/10 shadow-none hover:bg-foreground/15",
  leave: "",
};

export const VoiceControlButton = ({ label, icon: Icon, look, pressed, onClick }: VoiceControlButtonProps) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <Button
        variant={look === "leave" ? "destructive" : "outline"}
        size="icon"
        aria-label={label}
        aria-pressed={pressed}
        onClick={onClick}
        className={cn("size-9", LOOKS[look])}
      >
        <Icon className={cn("size-4", look === "plain" && "text-muted-foreground")} aria-hidden />
      </Button>
    </TooltipTrigger>
    <TooltipContent>{label}</TooltipContent>
  </Tooltip>
);
