import { XIcon } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { DeviceIcon } from "@/components/you/DeviceIcon";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { leavingRowClass } from "@/hooks/useRowGlide";
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

export const DeviceRow = ({ session, arrived = false, leaving = false, onSignOut }: DeviceRowProps) => {
  const name = [session.platform, session.label].filter(Boolean).join(" · ");
  return (
    <li
      className={cn(
        "group/row relative isolate flex min-h-14 items-center gap-3 py-2.5",
        leavingRowClass,
        arrived && "animate-in fade-in-0 slide-in-from-top-1 duration-800 ease-out",
      )}
      data-leaving={leaving || undefined}
    >
      {arrived && (
        <span
          aria-hidden
          className="pointer-events-none absolute inset-0 -z-10 bg-accent animate-out fade-out-0 delay-800 duration-[5600ms] ease-out fill-mode-forwards motion-reduce:hidden"
        />
      )}
      <DeviceIcon session={session} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{name}</p>
        <p className="flex min-w-0 items-center gap-1.5 text-xs text-muted-foreground">
          {session.current && <SettingsStatus tone="success">Active now</SettingsStatus>}
          {!session.current && <span className="tabular-nums">Last active {formatRelativeTime(session.last_active_at)}</span>}
          <span className="ml-2.5 truncate font-mono tabular-nums">{session.ip}</span>
        </p>
      </div>
      {session.current && <span className="text-xs font-medium text-foreground">This device</span>}
      {/* Quiet until the row is pointed at, so seven rows don't stack seven buttons; always shown where nothing hovers. */}
      {!session.current && onSignOut && (
        <span className="[@media(hover:hover)]:opacity-0 [@media(hover:hover)]:transition-opacity [@media(hover:hover)]:duration-150 [@media(hover:hover)]:group-focus-within/row:opacity-100 [@media(hover:hover)]:group-hover/row:opacity-100 [@media(hover:hover)]:has-data-[variant=destructive]:opacity-100">
        <ConfirmDestroyButton
          icon={XIcon}
          idleLabel="Sign out"
          label={`Sign out ${name}`}
          confirmLabel="Sign out"
          labelled
          onConfirm={onSignOut}
          loading={leaving}
        />
        </span>
      )}
    </li>
  );
};
