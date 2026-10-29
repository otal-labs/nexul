import { BoxIcon, LayersIcon } from "lucide-react";
import { Link } from "react-router";

import { RowContextMenu } from "@/components/listpane/RowContextMenu";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useStackPermissions } from "@/hooks/useStackPermissions";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { DeployStrategy, type ServiceDef } from "@/models/Service";

interface ServiceCardProps {
  service: ServiceDef;
}

// The machine it runs on, then the repository it builds from; the list endpoint carries no deploy image.
const sourceLine = (service: ServiceDef): string => {
  const repo = service.build_source;
  const built = repo?.repo_owner && repo?.repo_name ? `${repo.repo_owner}/${repo.repo_name}` : "";
  return [service.machine || service.target, built].filter(Boolean).join(" · ");
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
        meta={<span className="font-mono">{sourceLine(service)}</span>}
      />
    </RowContextMenu>
  );
};
