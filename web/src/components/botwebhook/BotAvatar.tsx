import { cn } from "@/lib/utils";

interface BotAvatarProps {
  avatar?: string | undefined;
  className?: string;
}

// A bot without an avatar of its own shows the Nexul glyph.
export const BotAvatar = ({ avatar, className }: BotAvatarProps) => (
  <img
    src={avatar || "/favicon.svg"}
    alt=""
    className={cn("size-8 shrink-0 rounded-full border border-border bg-accent object-cover", className)}
  />
);
