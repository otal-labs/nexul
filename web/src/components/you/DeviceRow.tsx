import { XIcon } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { DeviceIcon } from "@/components/you/DeviceIcon";
import { type ProtoDevice, usePrototypeStore } from "@/components/you/prototypeStore";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface DeviceRowProps {
  device: ProtoDevice;
}

export const DeviceRow = ({ device }: DeviceRowProps) => {
  const signOut = usePrototypeStore((s) => s.signOut);
  const remove = usePrototypeStore((s) => s.remove);
  const glow = usePrototypeStore((s) => s.variant === "flash");

  return (
    <li
      onAnimationEnd={(e) => e.target === e.currentTarget && device.leaving && remove(device.id)}
      className={cn(
        "relative isolate flex items-center gap-3 bg-card px-3 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40",
        device.arrived && glow && "animate-in fade-in-0 slide-in-from-top-1 duration-800 ease-out",
        device.arrived && !glow && "animate-in fade-in-0 slide-in-from-top-1 duration-200 ease-out",
        device.leaving && "animate-out fade-out-0 slide-out-to-top-1 duration-150 ease-standard fill-mode-forwards",
      )}
    >
      {device.arrived && glow && (
        <span
          aria-hidden
          className="pointer-events-none absolute inset-0 -z-10 bg-accent animate-out fade-out-0 delay-800 duration-[5600ms] ease-out fill-mode-forwards"
        />
      )}
      <DeviceIcon client={device.client} />
      <div className="relative min-w-0 flex-1">
        <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
          <span className="truncate">{device.label}</span>
          {device.current && (
            <span className="shrink-0 font-mono text-[10px] font-medium tracking-[0.14em] text-muted-foreground uppercase">
              This device
            </span>
          )}
        </p>
        <p className="truncate font-mono text-xs text-muted-foreground tabular-nums">
          {device.ip} · {device.current ? "active now" : formatRelativeTime(device.last_used_at)}
        </p>
      </div>
      {!device.current && (
        <ConfirmDestroyButton icon={XIcon} idleLabel="Sign out" confirmLabel="Sign out" onConfirm={() => signOut(device.id)} />
      )}
    </li>
  );
};
