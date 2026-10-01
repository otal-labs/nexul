import { TickerRow, type CheckOutcome } from "@/components/TickerRow";
import { useInstanceUpgrade } from "@/hooks/InstanceUpgradeHooks";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import { UpgradeRecordStatus, type InstanceUpgradeRecord } from "@/models/InstanceUpgrade";

// A failed poll mid-upgrade is the server restarting, so it moves the ticker on instead of showing an error.
const installState = (started: boolean, offline: boolean): CheckOutcome["state"] => {
  if (!started) return "idle";
  if (offline) return "ok";
  return "pending";
};

export const UpgradeElapsed = ({ record }: { record: InstanceUpgradeRecord }) => {
  const elapsed = useElapsedSeconds(Date.parse(record.created_at));

  return <span className="font-mono text-xs text-muted-foreground tabular-nums">{elapsed}s</span>;
};

export const InstanceUpgradeProgress = ({ record }: { record: InstanceUpgradeRecord }) => {
  const { error } = useInstanceUpgrade();
  const started = record.status === UpgradeRecordStatus.Started;
  const offline = Boolean(error);

  return (
    <ul className="space-y-3">
      <TickerRow
        label="Hand the upgrade to this machine"
        why="The instance runner starts nexul upgrade beside the server"
        outcome={{ state: started ? "ok" : "pending" }}
      />
      <TickerRow
        label={`Install ${record.to_version} and restart`}
        why="Nexul goes offline for a moment while its services restart"
        outcome={{ state: installState(started, offline) }}
      />
      <TickerRow
        label={`Come back on ${record.to_version}`}
        why="This page reconnects on its own, no refresh needed"
        outcome={{ state: offline ? "pending" : "idle" }}
      />
    </ul>
  );
};
