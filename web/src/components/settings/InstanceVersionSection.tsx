import { ArrowUpRight, RefreshCw } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Fact } from "@/components/Fact";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InstanceUpgradeProgress } from "@/components/settings/InstanceUpgradeProgress";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import {
  useInstanceUpgrade,
  useRefreshInstanceUpgrade,
  useRequestInstanceUpgrade,
} from "@/hooks/InstanceUpgradeHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { cn } from "@/lib/utils";
import { formatRelativeTime } from "@/utils/TimeUtility";
import { isUpgradeInProgress, UpgradeRecordStatus, type InstanceUpgrade } from "@/models/InstanceUpgrade";

// "dev build" gets a friendlier line than the raw reason string; every other reason already reads as one.
const reasonCopy = (reason: string): string =>
  reason === "dev build" ? "This build can't upgrade itself." : `Can't upgrade yet: ${reason}.`;

const NEWEST_RELEASE_REASON = "already on the newest release";

// The instance never moved, so the failure is still the story and the next click is a retry.
const isUnresolvedFailure = (data: InstanceUpgrade): boolean =>
  data.upgrade?.status === UpgradeRecordStatus.Failed && data.upgrade.to_version !== data.version;

const statusOf = (data: InstanceUpgrade): { headline: string; dot: string } => {
  if (data.upgrade && isUpgradeInProgress(data.upgrade)) {
    return { headline: `Upgrading to ${data.upgrade.to_version}`, dot: "bg-warning" };
  }
  if (data.upgrade && isUnresolvedFailure(data)) {
    return { headline: `Upgrade to ${data.upgrade.to_version} failed`, dot: "bg-destructive" };
  }
  if (!data.latest) return { headline: "No release to compare against", dot: "bg-muted-foreground" };
  if (data.update_available) return { headline: `${data.latest.version} is available`, dot: "bg-foreground" };
  return { headline: "Up to date", dot: "bg-success" };
};

const InstanceVersionFacts = ({ data }: { data: InstanceUpgrade }) => {
  const { headline, dot } = statusOf(data);

  return (
    <dl className="grid grid-cols-3 gap-x-6 gap-y-4 @xl:grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))]">
      <div className="col-span-3 @xl:col-span-1">
        <Fact label="Status">
          <span aria-hidden className={cn("size-2 shrink-0 rounded-full", dot)} />
          <span className="font-medium">{headline}</span>
        </Fact>
      </div>
      <Fact label="Running">
        <span className="font-mono break-all">{data.version}</span>
      </Fact>
      <Fact label="Channel">{data.channel}</Fact>
      <Fact label="Newest">
        {!data.latest && <span className="text-muted-foreground">None</span>}
        {data.latest && (
          <a
            href={data.latest.url}
            target="_blank"
            rel="noreferrer"
            aria-label={`Release notes for ${data.latest.version}`}
            className="inline-flex min-w-0 items-center gap-1 font-mono underline-offset-2 hover:underline"
          >
            <span className="break-all">{data.latest.version}</span>
            <ArrowUpRight className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </a>
        )}
      </Fact>
    </dl>
  );
};

const InstanceVersionBody = ({ data }: { data: InstanceUpgrade }) => {
  const record = data.upgrade;
  const inProgress = isUpgradeInProgress(record);
  const showReason =
    !data.can_upgrade && !inProgress && data.reason !== "" && data.reason !== NEWEST_RELEASE_REASON;

  return (
    <div className="@container space-y-6">
      <InstanceVersionFacts data={data} />
      {showReason && <p className="text-sm text-muted-foreground">{reasonCopy(data.reason)}</p>}
      {record && (inProgress || isUnresolvedFailure(data)) && <InstanceUpgradeProgress record={record} />}
      {record && record.status === UpgradeRecordStatus.Completed && (
        <p className="text-xs text-muted-foreground">
          Upgraded from <span className="font-mono">{record.from_version}</span> ·{" "}
          {formatRelativeTime(record.updated_at)}
        </p>
      )}
    </div>
  );
};

const CheckAgainButton = () => {
  const refresh = useRefreshInstanceUpgrade();

  return (
    <Button variant="ghost" size="sm" loading={refresh.isPending} onClick={() => refresh.mutate()}>
      <RefreshCw className="size-3.5" aria-hidden />
      Check again
    </Button>
  );
};

const upgradeLabel = (latest: InstanceUpgrade["latest"], retry: boolean): string => {
  if (retry) return "Try again";
  if (latest) return `Upgrade to ${latest.version}`;
  return "Upgrade";
};

const UpgradeButton = ({ latest, retry }: { latest: InstanceUpgrade["latest"]; retry: boolean }) => {
  const requestUpgrade = useRequestInstanceUpgrade();
  const { open: confirm } = useConfirmationDialog();

  const onUpgradeClick = async () => {
    const ok = await confirm({
      title: latest ? `Upgrade to ${latest.version}?` : "Upgrade this instance?",
      message: "Nexul downloads the release and restarts its services. The app reconnects when it's back.",
      confirmLabel: "Upgrade",
      destructive: false,
    });
    if (ok) requestUpgrade.mutate();
  };

  return (
    <Button size="sm" onClick={() => void onUpgradeClick()} loading={requestUpgrade.isPending}>
      {upgradeLabel(latest, retry)}
    </Button>
  );
};

const InstanceVersionFooter = ({ data }: { data: InstanceUpgrade }) => (
  <>
    <CheckAgainButton />
    {data.can_upgrade && <UpgradeButton latest={data.latest} retry={isUnresolvedFailure(data)} />}
  </>
);

export const InstanceVersionSection = () => {
  const { data, isPending, error } = useInstanceUpgrade();

  return (
    <SettingsCard
      id="instance-version"
      title="Instance version"
      footer={data && !isUpgradeInProgress(data.upgrade) && <InstanceVersionFooter data={data} />}
    >
      {isPending && <LoadingDisplay />}
      {error && !isUpgradeInProgress(data?.upgrade ?? null) && <ErrorDisplay error={error} />}
      {data && <InstanceVersionBody data={data} />}
    </SettingsCard>
  );
};
