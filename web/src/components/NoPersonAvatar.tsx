import { UserRound } from "lucide-react";

import { cn } from "@/lib/utils";

interface NoPersonAvatarProps {
  className?: string;
  label?: string;
}

// The face for a role with no one in it: a quiet disc, so an empty slot never reads as a missing picture.
export const NoPersonAvatar = ({ className, label }: NoPersonAvatarProps) => (
  <span title={label} className={cn("flex size-5 shrink-0 items-center justify-center rounded-full bg-muted-foreground/20 text-muted-foreground", className)}>
    <UserRound className="size-3/4" strokeWidth={2.25} aria-hidden />
  </span>
);
