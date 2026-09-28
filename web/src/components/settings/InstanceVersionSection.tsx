import { RefreshCw } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { Fact } from "@/components/Fact";
import {
  useInstanceUpgrade,
  useRefreshInstanceUpgrade,
  useRequestInstanceUpgrade,
} from "@/hooks/InstanceUpgradeHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { formatRelativeTime } from "@/utils/TimeUtility";
import { isUpgradeInProgress, type InstanceUpgrade } from "@/models/InstanceUpgrade";

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
      <CheckForReleaseButton />
    </Fact>
  </dl>
);

const CheckForReleaseButton = () => {
  const refresh = useRefreshInstanceUpgrade();

  return (
    <Button
      variant="ghost"
      size="icon"
      className="size-6 text-muted-foreground hover:text-foreground"
      aria-label="Check for a newer release"
      title="Check for a newer release"
      disabled={refresh.isPending}
      onClick={() => refresh.mutate()}
    >
      <RefreshCw
        className={refresh.isPending ? "size-3.5 animate-spin motion-reduce:animate-none" : "size-3.5"}
        aria-hidden
      />
    </Button>
  );
};

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
            On a Linux host, run <code className="font-mono">journalctl -u &apos;nexul-upgrade-*&apos;</code> for the
            upgrade&apos;s output; on a Mac or Windows PC it is in <code className="font-mono">nexul-upgrade.log</code>{" "}
            in Nexul&apos;s folder.
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
