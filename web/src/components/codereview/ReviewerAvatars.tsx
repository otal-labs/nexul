import { useState } from "react";

interface ReviewerAvatarsProps {
  names: string[];
}

// Names are GitHub logins (same as AssigneeAvatar); per-avatar failed state so one 404 doesn't degrade the whole stack.
const AvatarFace = ({ name }: { name: string }) => {
  const [failed, setFailed] = useState(false);

  return (
    <>
      {failed && name.charAt(0).toUpperCase()}
      {!failed && (
        <img
          src={`https://github.com/${name}.png`}
          alt=""
          referrerPolicy="no-referrer"
          onError={() => setFailed(true)}
          className="size-full rounded-full object-cover"
        />
      )}
    </>
  );
};

// Each avatar is independently focusable (aria-label carries the name), so the name popover works via keyboard too.
export const ReviewerAvatars = ({ names }: ReviewerAvatarsProps) => {
  const avatars = names.filter((name) => name.trim().length > 0);
  if (avatars.length === 0) return null;

  return (
    <div
      role="group"
      aria-label={`${avatars.length} reviewer${avatars.length === 1 ? "" : "s"}`}
      className="flex items-center -space-x-2"
    >
      {avatars.map((name, index) => (
        <span key={`${name}-${index}`} className="group/avatar relative">
          <span
            tabIndex={0}
            aria-label={name}
            className="flex size-5 items-center justify-center overflow-hidden rounded-full bg-secondary text-[9px] font-semibold text-secondary-foreground ring-2 ring-background transition-transform duration-150 ease-out hover:z-10 hover:scale-110 focus-visible:z-10 focus-visible:scale-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <AvatarFace name={name} />
          </span>
          <span
            aria-hidden="true"
            className="pointer-events-none absolute -top-8 left-1/2 z-20 -translate-x-1/2 translate-y-1 scale-95 rounded-md border border-border bg-popover px-2 py-1 text-xs whitespace-nowrap text-popover-foreground opacity-0 shadow-elevated transition-all duration-150 ease-out group-hover/avatar:translate-y-0 group-hover/avatar:scale-100 group-hover/avatar:opacity-100 group-focus-within/avatar:translate-y-0 group-focus-within/avatar:scale-100 group-focus-within/avatar:opacity-100"
          >
            {name}
          </span>
        </span>
      ))}
    </div>
  );
};
