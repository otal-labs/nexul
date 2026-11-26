import { BoxIcon, ChevronRightIcon, LayersIcon } from "lucide-react";
import { Link } from "react-router";

import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { formatRelativeTime } from "@/components/service/DeployTime";
import { HealthDot } from "@/components/service/HealthDot";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useFetchServiceDeploys } from "@/hooks/ServiceHooks";
import { useStackPermissions } from "@/hooks/useStackPermissions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { DeployStrategy, latestDeploy, type ServiceDef } from "@/models/Service";

interface ServiceCardProps {
  service: ServiceDef;
}

// The machine it runs on, then the repository it builds from, each named, since two mono ids side by side read as one.
const SourceLine = ({ service }: ServiceCardProps) => {
  const repo = service.build_source;
  const built = repo?.repo_owner && repo?.repo_name ? `${repo.repo_owner}/${repo.repo_name}` : "";
  const machine = service.machine || service.target;
  return (
    <span className="flex min-w-0 items-baseline gap-3">
      {machine && (
        <span className="truncate">
          on <span className="font-mono">{machine}</span>
        </span>
      )}
      {built && (
        <span className="truncate">
          from <span className="font-mono">{built}</span>
        </span>
      )}
    </span>
  );
};

// The newest deploy's state and age, from the query the stack page reads too.
const LastDeploy = ({ serviceId }: { serviceId: string }) => {
  const { data: deploys } = useFetchServiceDeploys(serviceId);
  const latest = latestDeploy(deploys);
  return (
    <>
      {deploys && !latest && "Never deployed"}
      {latest && (
        <span className="inline-flex items-center gap-1.5">
          <HealthDot status={latest.status} className="size-1.5 animate-none" />
          <span className="sr-only">{latest.status},</span>
          deployed {formatRelativeTime(latest.created_at)}
        </span>
      )}
    </>
  );
};

// One service, opening its stack; a right-click opens the stack's Permissions… for whoever may set its rules.
export const ServiceCard = ({ service }: ServiceCardProps) => {
  const canOpen = useAreaAccess()?.("stacks") ?? false;
  const { menu, manages } = useStackPermissions(service);
  const wsPath = useWorkspacePath();
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
        meta={<SourceLine service={service} />}
        count={canOpen && <LastDeploy serviceId={service.id} />}
        control={canOpen && <ChevronRightIcon className="pointer-events-none size-4 text-muted-foreground" aria-hidden />}
      />
    </RowContextMenu>
  );
};
