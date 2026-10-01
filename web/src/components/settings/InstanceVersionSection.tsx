import { RefreshCw } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { InstanceUpgradeProgress, UpgradeElapsed } from "@/components/settings/InstanceUpgradeProgress";
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
import { isUpgradeInProgress, type InstanceUpgrade } from "@/models/InstanceUpgrade";

// "dev build" gets a friendlier line than the raw reason string; every other reason already reads as one.
const reasonCopy = (reason: string): string =>
  reason === "dev build" ? "This build cannot upgrade itself." : reason;

const NEWEST_RELEASE_REASON = "already on the newest release";

const statusOf = (data: InstanceUpgrade): { headline: string; dot: string } => {
  if (data.upgrade && isUpgradeInProgress(data.upgrade)) {
    return { headline: `Upgrading to ${data.upgrade.to_version}`, dot: "bg-warning" };
  }
  if (!data.latest) return { headline: "No release to compare against", dot: "bg-muted-foreground" };
  if (data.update_available) return { headline: `${data.latest.version} is available`, dot: "bg-foreground" };
  return { headline: "Up to date", dot: "bg-success" };
};

const InstanceVersionStatus = ({ data }: { data: InstanceUpgrade }) => {
  const { headline, dot } = statusOf(data);
  const showReason =
    !data.can_upgrade && !isUpgradeInProgress(data.upgrade) && data.reason !== "" && data.reason !== NEWEST_RELEASE_REASON;

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between gap-2">
        <p className="flex items-center gap-2 text-base font-semibold">
          <span aria-hidden className={cn("size-2 shrink-0 rounded-full", dot)} />
          {headline}
        </p>
        {data.upgrade && isUpgradeInProgress(data.upgrade) && <UpgradeElapsed record={data.upgrade} />}
      </div>
      <p className="text-sm text-muted-foreground">
        Running <span className="font-mono">{data.version}</span> on the {data.channel} channel
        {data.latest && (
          <>
            {" · "}
            <a
              href={data.latest.url}
              target="_blank"
              rel="noreferrer"
              className="whitespace-nowrap underline-offset-2 hover:text-foreground hover:underline"
            >
              Release notes
            </a>
          </>
        )}
      </p>
      {showReason && <p className="text-xs text-muted-foreground">{reasonCopy(data.reason)}</p>}
    </div>
  );
};

const InstanceVersionBody = ({ data }: { data: InstanceUpgrade }) => {
  const record = data.upgrade;
  const inProgress = isUpgradeInProgress(record);

  return (
    <div className="space-y-4">
      <InstanceVersionStatus data={data} />
      {record && inProgress && <InstanceUpgradeProgress record={record} />}
      {record && !inProgress && record.status === "completed" && (
        <p className="text-xs text-muted-foreground">
          Upgraded to {record.to_version} · {formatRelativeTime(record.updated_at)}
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

const UpgradeButton = ({ latest }: { latest: InstanceUpgrade["latest"] }) => {
  const requestUpgrade = useRequestInstanceUpgrade();
  const { open: confirm } = useConfirmationDialog();

  const onUpgradeClick = async () => {
    const ok = await confirm({
      title: latest ? `Upgrade to ${latest.version}?` : "Upgrade this instance?",
      message: "This pulls the release's images, restarts the stack, and the app reconnects once it's back.",
      confirmLabel: "Upgrade",
      destructive: false,
    });
    if (ok) requestUpgrade.mutate();
  };

  return (
    <Button size="sm" onClick={() => void onUpgradeClick()} loading={requestUpgrade.isPending}>
      {latest ? `Upgrade to ${latest.version}` : "Upgrade"}
    </Button>
  );
};

const InstanceVersionFooter = ({ data }: { data: InstanceUpgrade }) => (
  <>
    <CheckAgainButton />
    {data.can_upgrade && <UpgradeButton latest={data.latest} />}
  </>
);

export const InstanceVersionSection = () => {
  const { data, isPending, error } = useInstanceUpgrade();

  return (
    <SettingsCard
      id="instance-version"
      title="Instance version"
      description="What this instance is running, and the newest release on its channel."
      footer={data && !isUpgradeInProgress(data.upgrade) && <InstanceVersionFooter data={data} />}
    >
      {isPending && <LoadingDisplay />}
      {error && !isUpgradeInProgress(data?.upgrade ?? null) && <ErrorDisplay error={error} />}
      {data && <InstanceVersionBody data={data} />}
    </SettingsCard>
  );
};
