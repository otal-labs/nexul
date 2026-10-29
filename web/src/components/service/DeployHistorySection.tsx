import { useState } from "react";

import { Button } from "@/components/ui/button";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { EmptyRow } from "@/components/EmptyRow";
import { EnterList } from "@/components/EnterList";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { DeployRow } from "@/components/service/DeployRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { settingsListClass } from "@/components/settings/SettingsRow";
import type { Deploy } from "@/models/Stack";
import { cn } from "@/lib/utils";

interface DeployHistorySectionProps {
  deploys: Deploy[] | undefined;
  lastHealthy: Deploy | undefined;
  isLoading: boolean;
  error?: Error | null;
}

// A screen of rows; the rest wait behind one button, since the newest are the ones read after a deploy.
const PAGE = 25;

type Show = "all" | "failed";

// Each DeployRow draws its own status, so a live update never triggers a full-row re-entrance.
export const DeployHistorySection = ({ deploys, lastHealthy, isLoading, error = null }: DeployHistorySectionProps) => {
  const [show, setShow] = useState<Show>("all");
  const [limit, setLimit] = useState(PAGE);
  const shown = (deploys ?? []).filter((d) => show === "all" || d.status === "failed");
  const older = shown.length - limit;
  return (
    <SettingsCard
      id="deploy-history"
      title="Deploys"
      aside={
        !!deploys?.length && (
          <ToggleGroup type="single" variant="segmented" size="xs" value={show} onValueChange={(value) => value && setShow(value as Show)} aria-label="Show">
            <ToggleGroupItem value="all">All</ToggleGroupItem>
            <ToggleGroupItem value="failed">Failed</ToggleGroupItem>
          </ToggleGroup>
        )
      }
    >
      {isLoading && <LoadingDisplay label="Loading deploys" />}
      {error && <ErrorDisplay error={error} />}
      {!isLoading && !error && (!deploys || deploys.length === 0) && <EmptyRow flush>No deploys yet. Deploy the stack and each run shows here.</EmptyRow>}
      {deploys && deploys.length > 0 && shown.length === 0 && <EmptyRow flush>No failed deploys.</EmptyRow>}
      {shown.length > 0 && (
        // Pulled out by the rows' own padding, so a row's hover and focus fill exactly the space between its hairlines.
        <EnterList className={cn(settingsListClass, "-mx-2")}>
          {shown.slice(0, limit).map((deploy) => (
            <DeployRow key={deploy.id} deploy={deploy} current={deploy.id === lastHealthy?.id} />
          ))}
        </EnterList>
      )}
      {older > 0 && (
        <Button type="button" variant="ghost" size="sm" className="mt-2 -ml-2 text-muted-foreground" onClick={() => setLimit(shown.length)}>
          {`Show ${older} older`}
        </Button>
      )}
    </SettingsCard>
  );
};
