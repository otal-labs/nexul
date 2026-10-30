import type { LucideIcon } from "lucide-react";

interface VoiceLobbyIconProps {
  icon: LucideIcon;
  className?: string;
}

// A lobby row's leading icon in an avatar-sized box, so it sits where the avatar stack does.
export const VoiceLobbyIcon = ({ icon: Icon, className = "size-4 text-muted-foreground" }: VoiceLobbyIconProps) => (
  <span className="flex size-7 shrink-0 items-center justify-center">
    <Icon className={className} aria-hidden />
  </span>
);
