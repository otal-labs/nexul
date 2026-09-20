import { useState } from "react";

import { cn, initials } from "@/lib/utils";

interface AssigneeAvatarProps {
  login: string;
  className?: string;
}

// github.com/<login>.png resolves the avatar with no API round-trip; the initials circle covers 404s.
export const AssigneeAvatar = ({ login, className }: AssigneeAvatarProps) => {
  const [failed, setFailed] = useState(false);

  if (failed) {
    return (
      <span
        title={login}
        className={cn(
          "flex size-5 shrink-0 items-center justify-center rounded-full bg-accent text-[9px] font-semibold text-accent-foreground",
          className,
        )}
      >
        {initials(login)}
      </span>
    );
  }

  return (
    <img
      src={`https://github.com/${login}.png`}
      alt=""
      title={login}
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
      className={cn("size-5 shrink-0 rounded-full bg-accent object-cover", className)}
    />
  );
};
