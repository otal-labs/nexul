import type { ReactNode } from "react";
import { ArrowUpRightIcon } from "lucide-react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { microheaderClass } from "@/components/Microheader";
import { MonoRef } from "@/components/MonoRef";
import { useFetchStackRequests } from "@/hooks/StackHooks";
import type { Stack } from "@/models/Stack";
import { cn } from "@/lib/utils";
import { errorRateOf } from "@/utils/ErrorRateUtility";
import { formatShare, requestTotals } from "@/utils/RequestsUtility";

export const HeroCell = ({ label, children }: { label: string; children: ReactNode }) => (
  <div className="min-w-0">
    <p className={microheaderClass}>{label}</p>
    <div className="mt-1.5 min-w-0 space-y-0.5">{children}</div>
  </div>
);

export const HeroLine = ({ dot, title, children }: { dot?: string; title?: string; children: ReactNode }) => (
  <p className="flex min-w-0 items-center gap-2 text-sm font-medium">
    {dot && <span aria-hidden className={cn("size-2 shrink-0 rounded-full", dot)} />}
    <span className="min-w-0 truncate" title={title}>{children}</span>
  </p>
);

export const HeroDetail = ({ indent = false, children }: { indent?: boolean; children: ReactNode }) => (
  <p className={cn("flex min-w-0 font-mono text-xs text-balance text-muted-foreground tabular-nums", indent && "pl-4")}>{children}</p>
);

const HostLink = ({ hostname, className }: { hostname: string; className?: string }) => (
  <a
    href={`https://${hostname}`}
    target="_blank"
    rel="noreferrer"
    className={`group flex max-w-full min-w-0 items-center gap-1 font-mono text-sm hover:underline hover:underline-offset-2 ${className ?? ""}`}
  >
    <MonoRef value={hostname} kind="hostname" className="min-w-0 shrink" style={{ width: `${hostname.length}ch` }} />
    <ArrowUpRightIcon className="size-3 shrink-0 text-muted-foreground group-hover:text-foreground" aria-hidden />
  </a>
);

// The first hostname with its domain kept whole; the rest one click away rather than a count nobody can open.
export const StackHostnamesCell = ({ hostnames }: { hostnames: string[] }) => {
  const [first, ...more] = hostnames;
  return (
    <HeroCell label="Hostnames">
      {!first && <p className="text-sm text-muted-foreground">None</p>}
      {first && <HostLink hostname={first} />}
      {more.length > 0 && (
        <Popover>
          <PopoverTrigger className="rounded-sm font-mono text-xs text-muted-foreground hover:text-foreground hover:underline hover:underline-offset-2">
            {`+${more.length} more`}
          </PopoverTrigger>
          <PopoverContent align="start" className="w-auto max-w-md space-y-1.5">
            {hostnames.map((hostname) => (
              <HostLink key={hostname} hostname={hostname} />
            ))}
          </PopoverContent>
        </Popover>
      )}
    </HeroCell>
  );
};

// The hour's totals; the 5xx share names its window, since Health's 5xx line is the error-rate rule's shorter one.
export const StackRequestsCell = ({ stack }: { stack: Stack }) => {
  const { data } = useFetchStackRequests(stack.id);
  const totals = data?.counted ? requestTotals(data) : undefined;
  const hasData = !!totals && totals.counted > 0;
  return (
    <HeroCell label="Requests">
      {!totals && <HeroDetail>—</HeroDetail>}
      {totals && !hasData && <p className="text-sm text-muted-foreground">{errorRateOf(stack).enabled ? "No data yet" : "Not counted"}</p>}
      {totals && hasData && <HeroLine>{`${totals.requests.toLocaleString()} / h`}</HeroLine>}
      {totals && hasData && <HeroDetail>{totals.errors > 0 ? `5xx ${formatShare(totals.share)} · hour` : "No 5xx · hour"}</HeroDetail>}
    </HeroCell>
  );
};
