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

export const DeviceRow = ({ session, arrived = false, leaving = false, onSignOut }: DeviceRowProps) => (
  <li
    className={cn(
      "relative isolate flex items-center gap-3 bg-card px-3 py-3 hover:bg-accent/40",
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
      <p className="truncate text-sm font-medium">{[session.platform, session.label].filter(Boolean).join(" · ")}</p>
      <p className="flex min-w-0 items-center gap-2 text-xs text-muted-foreground">
        {session.current && <SettingsStatus tone="success">This device</SettingsStatus>}
        {!session.current && <span>Active {formatRelativeTime(session.last_active_at)}</span>}
        <span className="truncate font-mono tabular-nums">{session.ip}</span>
      </p>
    </div>
    {!session.current && onSignOut && (
      <ConfirmDestroyButton icon={XIcon} idleLabel="Sign out" confirmLabel="Sign out" onConfirm={onSignOut} loading={leaving} />
    )}
  </li>
);
