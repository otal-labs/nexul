import { XIcon } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { DeviceIcon } from "@/components/you/DeviceIcon";
import { microheaderClass } from "@/components/Microheader";
import type { Session } from "@/models/User";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface DeviceRowProps {
  session: Session;
  /** A phone that just connected: rises in over 800ms with a glow that fades over 5600ms after an 800ms hold. */
  arrived?: boolean;
  leaving?: boolean;
  onSignOut?: () => void;
}

export const DeviceRow = ({ session, arrived = false, leaving = false, onSignOut }: DeviceRowProps) => (
  <li
    className={cn(
      "relative isolate flex items-center gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40",
      arrived && "animate-in fade-in-0 slide-in-from-top-1 duration-800 ease-out",
      leaving && "animate-out fade-out-0 slide-out-to-top-1 duration-150 ease-standard fill-mode-forwards",
    )}
  >
    {arrived && (
      <span
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-accent animate-out fade-out-0 delay-800 duration-[5600ms] ease-out fill-mode-forwards motion-reduce:hidden"
      />
    )}
    <DeviceIcon session={session} />
    <div className="min-w-0 flex-1">
      <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
        <span className="truncate">{[session.platform, session.label].filter(Boolean).join(" · ")}</span>
        {session.current && (
          <span className={cn(microheaderClass, "shrink-0")}>
            This device
          </span>
        )}
      </p>
      <p className="truncate font-mono text-xs text-muted-foreground tabular-nums">
        {session.ip} · {session.current ? "active now" : formatRelativeTime(session.last_active_at)}
      </p>
    </div>
    {!session.current && onSignOut && (
      <ConfirmDestroyButton icon={XIcon} idleLabel="Sign out" confirmLabel="Sign out" onConfirm={onSignOut} loading={leaving} />
    )}
  </li>
);
