import { useState } from "react";
import { LogOut } from "lucide-react";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { DeviceRow } from "@/components/you/DeviceRow";
import { useListSessions, useSignOutOtherSessions, useSignOutSession } from "@/hooks/AuthHooks";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";

export const DevicesFeed = () => {
  const { data, error, isPending } = useListSessions();
  const signOut = useSignOutSession();
  const signOutOthers = useSignOutOtherSessions();
  // Rows exit while the request is in flight; the refetch removes them, a failure brings them back.
  const [leaving, setLeaving] = useState<string[]>([]);
  // A phone that signed in after this list appeared gets the arrival rise and glow; one already listed does not.
  const [mountedAt] = useState(() => Date.now());
  const arrivals = useDeviceArrivalStore((s) => s.arrivals);
  const arrived = (id: string) => arrivals.some((a) => a.id === id && a.at >= mountedAt);

  const current = data?.sessions.filter((session) => session.current) ?? [];
  const others = data?.sessions.filter((session) => !session.current) ?? [];

  const signOutOne = (id: string) => {
    setLeaving((ids) => [...ids, id]);
    signOut.mutate(id, { onError: () => setLeaving((ids) => ids.filter((left) => left !== id)) });
  };

  const signOutAll = () => {
    setLeaving(others.map((session) => session.id));
    signOutOthers.mutate(undefined, { onError: () => setLeaving([]) });
  };

  return (
    <SettingsCard
      id="devices"
      title="Signed-in devices"
      description="Sign out any you don't recognise."
      footer={
        <>
          <p className="text-sm text-muted-foreground">Every other device will have to sign in again.</p>
          <Button
            variant="destructive"
            size="sm"
            loading={signOutOthers.isPending}
            disabled={others.length === 0}
            onClick={signOutAll}
          >
            <LogOut className="size-4" aria-hidden />
            Sign out everywhere else
          </Button>
        </>
      }
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && (
        <div className="space-y-6">
          <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
            {current.map((session) => (
              <DeviceRow key={session.id} session={session} />
            ))}
          </ul>
          <div className="space-y-2">
            <h3 className="text-xs font-medium text-muted-foreground">Other devices</h3>
            {others.length === 0 && (
              <EmptyRow className="animate-in fade-in-0 duration-200 ease-out">No other devices signed in.</EmptyRow>
            )}
            {others.length > 0 && (
              <ul className="divide-y divide-border overflow-hidden rounded-md border border-border">
                {others.map((session) => (
                  <DeviceRow
                    key={session.id}
                    session={session}
                    arrived={arrived(session.id)}
                    leaving={leaving.includes(session.id)}
                    onSignOut={() => signOutOne(session.id)}
                  />
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </SettingsCard>
  );
};
