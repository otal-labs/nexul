import { Link } from "react-router";

import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { RequestBars } from "@/components/stack/RequestBars";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchStackRequests } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Stack, StackRequests } from "@/models/Stack";
import { errorRateOf, ruleText } from "@/utils/ErrorRateUtility";
import { formatShare, missingSince, requestTotals } from "@/utils/RequestsUtility";
import { formatClockTime } from "@/utils/TimeUtility";

interface RequestsSectionProps {
  stack: Stack;
}

const Summary = ({ requests }: { requests: StackRequests }) => {
  const { requests: total, errors, share } = requestTotals(requests);
  const since = missingSince(requests);
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-1">
        <p className="font-mono text-sm tabular-nums">
          <span className="text-2xl font-medium tracking-tight">{total.toLocaleString()}</span>{" "}
          <span className="text-muted-foreground">requests in the last hour</span>
        </p>
        <p className="flex items-center gap-2 font-mono text-sm tabular-nums">
          {errors > 0 && <span aria-hidden className="size-2 shrink-0 rounded-full bg-warning" />}
          {errors > 0 && <span>{`5xx ${formatShare(share)}`}</span>}
          {errors > 0 && <span className="text-muted-foreground">{`(${errors.toLocaleString()})`}</span>}
          {errors === 0 && <span className="text-muted-foreground">No 5xx</span>}
        </p>
      </div>
      <div className="rounded-lg bg-surface-2 p-3 ring-1 ring-border">
        <RequestBars requests={requests} />
        <div className="mt-2 flex justify-between font-mono text-xs text-muted-foreground">
          <span>60 min ago</span>
          <span>now</span>
        </div>
      </div>
      {since && <p className="text-xs text-muted-foreground">{`No request data since ${formatClockTime(since)}`}</p>}
    </div>
  );
};

// A stack with a hostname: what its hostnames served in the last hour, counted a minute at a time (nothing per request is kept).
// The footer states the error-rate rule, since requests are only counted for a stack that has it on.
export const RequestsSection = ({ stack }: RequestsSectionProps) => {
  const { data, error } = useFetchStackRequests(stack.id);
  const wsPath = useWorkspacePath();
  const rule = errorRateOf(stack);
  const requests = data?.counted ? data : undefined;
  const hasData = !!requests && requestTotals(requests).counted > 0;
  return (
    <>
      {error && (
        <SettingsCard id="requests" title="Requests">
          <ErrorDisplay error={error} title="Couldn't load requests." />
        </SettingsCard>
      )}
      {requests && (
        <SettingsCard
          id="requests"
          title="Requests"
          description="What this stack's hostnames served in the last hour, counted a minute at a time."
          footer={rule.enabled && <p className="text-xs text-muted-foreground">{`Alert when ${ruleText(rule)}.`}</p>}
        >
          {hasData && <Summary requests={requests} />}
          {!hasData && rule.enabled && <EmptyRow flush>No request data yet. Counts show up within a few minutes of the first request.</EmptyRow>}
          {!hasData && !rule.enabled && (
            <EmptyRow flush>
              Not counted. Turn on the error-rate rule in{" "}
              <Link to={wsPath(`/stacks/${stack.id}/alerts`)} className="text-brand underline-offset-4 hover:underline">
                Alerts
              </Link>
              .
            </EmptyRow>
          )}
        </SettingsCard>
      )}
    </>
  );
};
