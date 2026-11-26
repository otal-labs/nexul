import { deployOutcome } from "@nexul/client-core/deployOutcome";
import { BoxIcon, ChevronRightIcon, LayersIcon } from "lucide-react";
import { Link } from "react-router";

import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { SettingsStatus, type SettingsStatusTone } from "@/components/settings/SettingsStatus";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchStackDeploys } from "@/hooks/StackHooks";
import { useStackPermissions } from "@/hooks/useStackPermissions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { DeployStatus, DeployStrategy, type ServiceDef } from "@/models/Service";
import { latestDeploy } from "@/models/Stack";
import { formatRelativeTime } from "@/utils/TimeUtility";

interface ServiceCardProps {
  service: ServiceDef;
}

// The machine it runs on, then the repository it builds from; the list endpoint carries no deploy image.
const sourceLine = (service: ServiceDef): string => {
  const repo = service.build_source;
  const built = repo?.repo_owner && repo?.repo_name ? `${repo.repo_owner}/${repo.repo_name}` : "";
  return [service.machine || service.target, built].filter(Boolean).join(" · ");
};

const outcomeTone: Record<DeployStatus, SettingsStatusTone> = {
  [DeployStatus.Pending]: "warning",
  [DeployStatus.Running]: "info",
  [DeployStatus.Healthy]: "success",
  [DeployStatus.Failed]: "destructive",
};

// One service, opening its stack; a right-click opens the stack's Permissions… for whoever may set its rules.
export const ServiceCard = ({ service }: ServiceCardProps) => {
  const canOpen = useAreaAccess()?.("stacks") ?? false;
  const { menu, manages } = useStackPermissions(service);
  const wsPath = useWorkspacePath();
  // The stack page reads the same history, so opening a row lands on a warm cache.
  const { data: deploys } = useFetchStackDeploys(canOpen ? service.id : undefined);
  const latest = latestDeploy(deploys);
  const Strategy = service.strategy === DeployStrategy.Compose ? LayersIcon : BoxIcon;
  const name = <span className="truncate font-medium" title={service.name}>{service.name}</span>;
  return (
    <RowContextMenu menu={menu} manages={manages}>
      <SettingsListRow
        lead={<Strategy className="size-3.5 text-muted-foreground" role="img" aria-label={service.strategy} />}
        name={
          <>
            {canOpen && (
              <Link to={wsPath(`/stacks/${service.id}`)} className="min-w-0 truncate after:absolute after:inset-0">
                {name}
              </Link>
            )}
            {!canOpen && name}
          </>
        }
        meta={<span className="font-mono">{sourceLine(service)}</span>}
        control={
          canOpen && (
            <span className="pointer-events-none flex items-center gap-3">
              {latest && (
                <SettingsStatus tone={outcomeTone[latest.status]} detail={formatRelativeTime(latest.created_at)}>
                  {deployOutcome(latest.status)}
                </SettingsStatus>
              )}
              {deploys && !latest && <SettingsStatus tone="muted">Never deployed</SettingsStatus>}
              <ChevronRightIcon className="size-4 text-muted-foreground" aria-hidden />
            </span>
          )
        }
      />
    </RowContextMenu>
  );
};
