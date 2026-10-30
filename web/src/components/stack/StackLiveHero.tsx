import { deployOutcome } from "@nexul/client-core/deployOutcome";

import { HealthDot } from "@/components/service/HealthDot";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { MonoRef } from "@/components/MonoRef";
import { HeroCell, HeroDetail, HeroLine, StackHostnamesCell, StackRequestsCell } from "@/components/stack/StackHeroCells";
import { useFetchStackServices } from "@/hooks/StackHooks";
import { useNow } from "@/hooks/useNow";
import { cn } from "@/lib/utils";
import { isServiceDown, type Container, type Deploy, type StackWithBranches } from "@/models/Stack";
import { errorRateHealth, errorRateOf } from "@/utils/ErrorRateUtility";
import { elapsedShort, imageShortName } from "@/utils/ImageRefUtility";
import { formatClockTime } from "@/utils/TimeUtility";

interface StackLiveHeroProps {
  stack: StackWithBranches;
  latest: Deploy | undefined;
  lastHealthy: Deploy | undefined;
  hostnames: string[];
}

// A runner reports what it sees after each deploy and each container event; older than this, Health says how old it is.
const STALE_MS = 60 * 60_000;

// An outage whose service the runner no longer lists is still down, so its name is counted too.
const downNames = (services: Container[], stack: StackWithBranches): string[] => [
  ...new Set([...(stack.outages ?? []).map((o) => o.service), ...services.filter((c) => isServiceDown(c, stack.outages)).map((c) => c.name)]),
];

const newestObservation = (services: Container[]): string => services.map((c) => c.observed_at ?? "").reduce((a, b) => (a > b ? a : b), "");

const Health = ({ services, stack }: { services: Container[] | undefined; stack: StackWithBranches }) => {
  const names = services ? downNames(services, stack) : [];
  const total = services?.length ?? 0;
  const rate = errorRateHealth(stack);
  const observed = services ? newestObservation(services) : "";
  const now = useNow(false);
  const stale = !!observed && now - Date.parse(observed) > STALE_MS;
  return (
    <HeroCell label="Health">
      {!services && <HeroDetail>—</HeroDetail>}
      {services && names.length === 0 && !rate && <HeroLine dot="bg-success">No issues</HeroLine>}
      {services && names.length === 0 && !rate && <HeroDetail indent>{`${total} up`}</HeroDetail>}
      {names.length > 0 && (
        <HeroLine dot="bg-destructive" title={names.join(", ")}>
          {names.length === 1 ? `${names[0]} down` : `${names.length} of ${total} down`}
        </HeroLine>
      )}
      {/* The rule's own window, which the Requests cell's hour would otherwise contradict. */}
      {rate && (
        <HeroLine dot={rate.tone === "warning" ? "bg-warning-mark" : "bg-success"}>
          {rate.tone === "warning" ? `${rate.text} · ${errorRateOf(stack).minutes} min` : rate.text}
        </HeroLine>
      )}
      {rate && <HeroDetail indent>{`${rate.tone === "warning" ? "since" : "at"} ${formatClockTime(rate.at)}`}</HeroDetail>}
      {stale && <HeroDetail indent>{`services seen ${formatRelativeTime(observed)}`}</HeroDetail>}
    </HeroCell>
  );
};

// The newest attempt and its own facts, then the image that is live now, always marked "live" so it never reads as the attempt's.
const LastDeploy = ({ latest, lastHealthy }: { latest: Deploy | undefined; lastHealthy: Deploy | undefined }) => {
  const inFlight = latest?.status === "running" || latest?.status === "pending";
  const now = useNow(inFlight);
  const live = latest?.status === "healthy" ? latest : lastHealthy;
  const facts = [!inFlight && latest && formatRelativeTime(latest.created_at), latest?.triggered_by].filter(Boolean).join(" · ");
  return (
    <HeroCell label="Last deploy">
      {!latest && <p className="text-sm text-muted-foreground">No deploys yet</p>}
      {latest && (
        <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
          <HealthDot status={latest.status} className="shrink-0 animate-none" />
          {inFlight ? `${deployOutcome(latest.status)} · ${elapsedShort(latest.created_at, now)}` : deployOutcome(latest.status)}
        </p>
      )}
      {facts && <HeroDetail indent>{facts}</HeroDetail>}
      {live?.image && (
        <HeroDetail indent>
          <span className="mr-1.5 shrink-0">live</span>
          <MonoRef value={imageShortName(live.image)} kind="image" title={live.image} className="text-foreground" />
        </HeroDetail>
      )}
    </HeroCell>
  );
};

// What a stack is doing right now, read before anything else on its page: one strip, health first.
export const StackLiveHero = ({ stack, latest, lastHealthy, hostnames }: StackLiveHeroProps) => {
  const { data: services } = useFetchStackServices(stack.id);
  return (
    <div className="@container">
      <section
        aria-label="Live status"
        className={cn(
          "grid grid-cols-2 gap-x-6 gap-y-5 rounded-lg bg-surface-2 px-5 py-4 ring-1 ring-border",
          "@3xl:grid-cols-[minmax(0,1.1fr)_minmax(0,1.35fr)_auto_minmax(0,1.2fr)]",
        )}
      >
        <Health services={services} stack={stack} />
        <LastDeploy latest={latest} lastHealthy={lastHealthy} />
        <StackRequestsCell stack={stack} />
        <StackHostnamesCell hostnames={hostnames} />
      </section>
    </div>
  );
};
