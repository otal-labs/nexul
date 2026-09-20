import type { ReactNode } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { useInstanceUpgrade, useRequestInstanceUpgrade } from "@/hooks/InstanceUpgradeHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { formatRelativeTime } from "@/utils/TimeUtility";
import { isUpgradeInProgress, type InstanceUpgrade } from "@/models/InstanceUpgrade";

interface FactProps {
  label: string;
  children: ReactNode;
}

const Fact = ({ label, children }: FactProps) => (
  <div className="min-w-0">
    <dt className="font-mono text-[11px] font-medium tracking-[0.18em] text-muted-foreground uppercase">{label}</dt>
    <dd className="mt-1 text-sm">{children}</dd>
  </div>
);

const InstanceVersionFacts = ({ data }: { data: InstanceUpgrade }) => (
  <dl className="grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-3">
    <Fact label="Running">
      <span className="font-mono text-xs">{data.version}</span>
    </Fact>
    <Fact label="Channel">
      <span className="font-mono text-xs">{data.channel}</span>
    </Fact>
    <Fact label="Newest release">
      {data.latest && (
        <a
          href={data.latest.url}
          target="_blank"
          rel="noreferrer"
          className="font-mono text-xs underline underline-offset-2"
        >
          {data.latest.version}
        </a>
      )}
      {!data.latest && <span className="text-xs text-muted-foreground">—</span>}
    </Fact>
  </dl>
);

// "dev build" gets a friendlier line than the raw reason string; every other reason already reads as one.
const reasonCopy = (reason: string): string =>
  reason === "dev build" ? "This build cannot upgrade itself." : reason;

const InstanceVersionAction = ({ data }: { data: InstanceUpgrade }) => {
  const requestUpgrade = useRequestInstanceUpgrade();
  const { open: confirm } = useConfirmationDialog();

  const onUpgradeClick = async () => {
    const ok = await confirm({
      title: data.latest ? `Upgrade to ${data.latest.version}?` : "Upgrade this instance?",
      message: "This pulls the release's images, restarts the stack, and the app reconnects once it's back.",
      confirmLabel: "Upgrade",
      destructive: false,
    });
    if (ok) requestUpgrade.mutate();
  };

  return (
    <div className="space-y-1.5">
      <Button
        onClick={() => void onUpgradeClick()}
        disabled={!data.can_upgrade || requestUpgrade.isPending}
      >
        {data.latest ? `Upgrade to ${data.latest.version}` : "Upgrade"}
      </Button>
      {!data.can_upgrade && <p className="text-xs text-muted-foreground">{reasonCopy(data.reason)}</p>}
    </div>
  );
};

const InstanceVersionBody = ({ data }: { data: InstanceUpgrade }) => {
  const record = data.upgrade;
  const inProgress = isUpgradeInProgress(record);

  return (
    <div className="space-y-4">
      <InstanceVersionFacts data={data} />
      {record && inProgress && (
        <p className="text-sm text-muted-foreground">
          Upgrading to {record.to_version}… the app will reconnect on its own
        </p>
      )}
      {record && !inProgress && record.status === "completed" && (
        <p className="text-sm text-muted-foreground">
          Upgraded to {record.to_version} at {formatRelativeTime(record.updated_at)}
        </p>
      )}
      {record && !inProgress && record.status === "failed" && (
        <div className="space-y-1 rounded-md border border-destructive/30 bg-destructive/5 p-3">
          <p className="text-sm text-destructive">{record.error}</p>
          <p className="text-xs text-muted-foreground">
            On the host, run <code className="font-mono">docker logs nexul-upgrade</code> for the helper&apos;s
            output.
          </p>
        </div>
      )}
      {!inProgress && <InstanceVersionAction data={data} />}
    </div>
  );
};

export const InstanceVersionSection = () => {
  const { data, isPending, error } = useInstanceUpgrade();

  return (
    <SettingsCard
      id="instance-version"
      title="Instance version"
      description="What this instance is running, and the newest release on its channel."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <InstanceVersionBody data={data} />}
    </SettingsCard>
  );
};
