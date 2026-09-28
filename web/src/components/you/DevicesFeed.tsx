import { LogOut } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { DeviceRow } from "@/components/you/DeviceRow";
import { usePrototypeStore } from "@/components/you/prototypeStore";

export const DevicesFeed = () => {
  const devices = usePrototypeStore((s) => s.devices);
  const signOutOthers = usePrototypeStore((s) => s.signOutOthers);
  const current = devices.filter((device) => device.current);
  const others = devices.filter((device) => !device.current);

  return (
    <SettingsCard
      id="devices"
      title="Signed-in devices"
      description="Every browser, desktop app, and phone signed in to your account. Sign out any you don't recognise."
      footer={
        <>
          <p className="text-sm text-muted-foreground">Everything except this device will need to sign in again.</p>
          <Button variant="destructive" size="sm" disabled={others.length === 0} onClick={signOutOthers}>
            <LogOut className="size-4" aria-hidden />
            Sign out everywhere else
          </Button>
        </>
      }
    >
      <div className="space-y-6">
        <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
          {current.map((device) => (
            <DeviceRow key={device.id} device={device} />
          ))}
        </ul>
        <div className="space-y-2">
          <h3 className="text-xs font-medium text-muted-foreground">Other devices</h3>
          {others.length === 0 && (
            <EmptyRow className="animate-in fade-in-0 duration-200 ease-out">No other devices are signed in.</EmptyRow>
          )}
          {others.length > 0 && (
            <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
              {others.map((device) => (
                <DeviceRow key={device.id} device={device} />
              ))}
            </ul>
          )}
        </div>
      </div>
    </SettingsCard>
  );
};
