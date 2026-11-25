import { healthFigures, type PlayRunHealth } from "@nexul/client-core/plays";

import { RelativeTime } from "@/components/RelativeTime";
import { Text } from "@/components/ui/text";

// The web's run health in one mono line: busiest first, then the week, then the last run's age while nothing runs.
export const PlayHealthLine = ({ health }: { health: PlayRunHealth }) => (
  <Text numberOfLines={2} className="font-mono text-xs text-muted-foreground">
    {!health.last_run && "Not run yet"}
    {health.last_run && healthFigures(health).map((figure) => figure.text).join(" · ")}
    {health.last_run && health.running === 0 && (
      <>
        {" · last "}
        <RelativeTime iso={health.last_run.started_at} />
        {" ago"}
      </>
    )}
  </Text>
);
