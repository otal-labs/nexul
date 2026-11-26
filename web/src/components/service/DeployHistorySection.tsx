import { useState } from "react";

import { Button } from "@/components/ui/button";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DeployRow } from "@/components/service/DeployRow";
import { useDeployFilter } from "@/hooks/useDeployFilter";
import type { Deploy } from "@/models/Stack";

interface DeployHistorySectionProps {
  deploys: Deploy[] | undefined;
  lastHealthy: Deploy | undefined;
  isLoading: boolean;
  error?: Error | null;
}

// A screen of rows; the rest wait behind one button, since the newest are the ones read after a deploy.
const PAGE = 25;

// The Deploys tab names the list, so the card carries no title; its All / Failed sits on the tab row (DeployFilterToggle).
// Each DeployRow draws its own status, so a live update never triggers a full-row re-entrance.
export const DeployHistorySection = ({ deploys, lastHealthy, isLoading, error = null }: DeployHistorySectionProps) => {
  const [show] = useDeployFilter();
  const [limit, setLimit] = useState(PAGE);
  const shown = (deploys ?? []).filter((d) => show === "all" || d.status === "failed");
  const older = shown.length - limit;
  return (
    <section id="deploy-history" aria-label="Deploys" className="rounded-lg border border-border bg-card px-6 py-3 shadow-card">
      {isLoading && <LoadingDisplay label="Loading deploys" />}
      {error && <ErrorDisplay error={error} />}
      {!isLoading && !error && (!deploys || deploys.length === 0) && <EmptyRow flush>No deploys yet. Deploy the stack and each run shows here.</EmptyRow>}
      {deploys && deploys.length > 0 && shown.length === 0 && <EmptyRow flush>No failed deploys.</EmptyRow>}
      {shown.length > 0 && (
        // Pulled out by the rows' own padding, so a row's hover and focus fill exactly the space between its hairlines.
        <EnterList className="-mx-2 divide-y divide-border">
          {shown.slice(0, limit).map((deploy, i) => (
            <DeployRow key={deploy.id} deploy={deploy} current={deploy.id === lastHealthy?.id} aboveTriggeredBy={shown[i - 1]?.triggered_by} />
          ))}
        </EnterList>
      )}
      {older > 0 && (
        <Button type="button" variant="ghost" size="sm" className="mt-2 -ml-2 text-muted-foreground" onClick={() => setLimit(shown.length)}>
          {`Show ${older} older`}
        </Button>
      )}
    </section>
  );
};
