import { DeployStepList } from "@/components/deploy/DeployStepList";
import { useInstanceUpgrade } from "@/hooks/InstanceUpgradeHooks";
import { useNow } from "@/hooks/useNow";
import { isUpgradeInProgress, UpgradeRecordStatus, type InstanceUpgradeRecord } from "@/models/InstanceUpgrade";
import type { DeployStepState } from "@/utils/DeployLogUtility";

interface UpgradePhase {
  key: string;
  label: string;
  state: DeployStepState;
  durationMs: number | undefined;
}

// A failed poll mid-upgrade is the server restarting, so it moves the phases on instead of showing an error.
const phaseStates = (record: InstanceUpgradeRecord, offline: boolean): DeployStepState[] => {
  if (record.status === UpgradeRecordStatus.Failed) return ["done", "failed", "pending"];
  if (offline) return ["done", "done", "active"];
  if (record.status === UpgradeRecordStatus.Started) return ["done", "active", "pending"];
  return ["active", "pending", "pending"];
};

const upgradePhases = (record: InstanceUpgradeRecord, offline: boolean, now: number): UpgradePhase[] => {
  const labels = ["Hand off to this machine", `Download and install ${record.to_version}`, `Restart on ${record.to_version}`];
  const started = Date.parse(record.created_at);
  const ranFor = record.status === UpgradeRecordStatus.Failed ? Date.parse(record.updated_at) - started : now - started;
  return phaseStates(record, offline).map((state, i) => ({
    key: String(i),
    label: labels[i] ?? "",
    state,
    durationMs: state === "active" || state === "failed" ? Math.max(0, ranFor) : undefined,
  }));
};

const UpgradeFailure = ({ error }: { error: string }) => (
  <div className="space-y-2 animate-in fade-in-0 slide-in-from-bottom-1 duration-200 ease-out">
    <div className="terminal-window min-w-0 overflow-hidden">
      <div className="terminal-window__bar">
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__title">nexul upgrade</span>
      </div>
      <pre role="alert" className="terminal-window__body text-xs leading-relaxed whitespace-pre-wrap break-words">
        {error}
      </pre>
    </div>
    <p className="text-xs text-muted-foreground">
      Full output on the server: <code className="font-mono">journalctl -u &apos;nexul-upgrade-*&apos;</code>, or{" "}
      <code className="font-mono">nexul-upgrade.log</code> in Nexul&apos;s folder on a Mac or Windows PC.
    </p>
  </div>
);

export const InstanceUpgradeProgress = ({ record }: { record: InstanceUpgradeRecord }) => {
  const { error } = useInstanceUpgrade();
  const now = useNow(isUpgradeInProgress(record));
  const phases = upgradePhases(record, Boolean(error), now);
  const failed = record.status === UpgradeRecordStatus.Failed;

  return (
    <div className="space-y-4">
      <DeployStepList
        title={failed ? `Upgrade to ${record.to_version} failed` : `Upgrading to ${record.to_version}`}
        steps={phases}
      />
      {failed && <UpgradeFailure error={record.error} />}
    </div>
  );
};
