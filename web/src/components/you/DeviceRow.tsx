import { XIcon } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { DeviceIcon } from "@/components/you/DeviceIcon";
import type { Session } from "@/models/User";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface DeviceRowProps {
  session: Session;
  leaving?: boolean;
  onSignOut?: () => void;
}

export const DeviceRow = ({ session, leaving = false, onSignOut }: DeviceRowProps) => (
  <li
    className={cn(
      "flex items-center gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40",
      leaving && "animate-out fade-out-0 slide-out-to-top-1 duration-150 ease-standard fill-mode-forwards",
    )}
  >
    <DeviceIcon session={session} />
    <div className="min-w-0 flex-1">
      <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
        <span className="truncate">{[session.platform, session.label].filter(Boolean).join(" · ")}</span>
        {session.current && (
          <span className="shrink-0 font-mono text-[10px] font-medium tracking-[0.14em] text-muted-foreground uppercase">
            This device
          </span>
        )}
      </p>
      <p className="truncate font-mono text-xs text-muted-foreground tabular-nums">
        {session.ip} · {session.current ? "active now" : formatRelativeTime(session.last_active_at)}
      </p>
    </div>
    {!session.current && onSignOut && (
      <ConfirmDestroyButton icon={XIcon} idleLabel="Sign out" confirmLabel="Sign out" onConfirm={onSignOut} disabled={leaving} />
    )}
  </li>
);
