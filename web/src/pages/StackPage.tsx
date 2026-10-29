import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ServiceHostnameSection } from "@/components/dns/ServiceHostnameSection";
import { DeployHistorySection } from "@/components/service/DeployHistorySection";
import { RequestsSection } from "@/components/stack/RequestsSection";
import { StackAlertsSection } from "@/components/stack/StackAlertsSection";
import { StackBranchDeploySection } from "@/components/stack/StackBranchDeploySection";
import { StackDangerZoneSection } from "@/components/stack/StackDangerZoneSection";
import { ServicesSection } from "@/components/stack/ServicesSection";
import { StackEnvironmentSection } from "@/components/stack/StackEnvironmentSection";
import { StackLogsSection } from "@/components/stack/StackLogsSection";
import { StackHeaderSection } from "@/components/stack/StackHeaderSection";
import { DEFAULT_STACK_SECTION, isStackSection, StackNav, type StackSection } from "@/components/stack/StackNav";
import { StackTabs, type StackTab } from "@/components/stack/StackTabs";
import { SettingsShell } from "@/components/settings/SettingsShell";
import { useAreaAccess, useHasInstancePermission } from "@/hooks/AccessHooks";
import { useFetchExposures } from "@/hooks/DnsHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchStack, useFetchStackDeploys, useFetchStackServices } from "@/hooks/StackHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { latestDeploy, type Container as StackContainer } from "@/models/Stack";
import { projectSettingsPath, projectTokenById } from "@/models/Project";

// A run stack's one container: its observed image, else the declared one. A compose stack has no single image.
const imageOf = (c: StackContainer | undefined): string | undefined => c?.image || c?.declared.image;

interface StackPageProps {
  // Each tab is its own route; Settings carries its section as the next path segment.
  tab?: StackTab;
}

export const StackPage = ({ tab = "overview" }: StackPageProps) => {
  const { stackId = "", section: pathSection, service } = useParams();
  const { data: stack, isPending, error } = useFetchStack(stackId);
  const { data: projects = [] } = useFetchProjects();
  const { data: services } = useFetchStackServices(stackId);
  const { data: deploys, isPending: deploysPending, error: deploysError } = useFetchStackDeploys(stackId);
  const can = useAreaAccess(stack?.project_id || undefined);
  const showExposures = useHasInstancePermission("dns:read");
  const showLogs = can?.("stackLogs") ?? false;
  const wsPath = useWorkspacePath();
  const { data: exposures } = useFetchExposures(showExposures);

  const requested: StackSection = isStackSection(pathSection) ? pathSection : DEFAULT_STACK_SECTION;
  const showBranches = !!stack && !stack.derived_from;
  const hidden = (requested === "branches" && !showBranches) || (requested === "exposures" && !showExposures);
  const section = hidden ? DEFAULT_STACK_SECTION : requested;

  const latest = latestDeploy(deploys);
  const lastHealthy = deploys?.find((d) => d.status === "healthy");
  const containerIds = new Set((services ?? []).map((c) => c.id));
  const hostnames = (exposures ?? [])
    .filter((e) => !!e.service_id && containerIds.has(e.service_id))
    .map((e) => e.hostname);
  // An instance stack (a gateway) has no project; Topology is where it lives.
  const projectPath = wsPath(stack?.project_id ? projectSettingsPath(projectTokenById(projects, stack.project_id)) : "/topology");
  const image = latest?.image || (stack?.strategy === "run" ? imageOf(services?.[0]) : undefined);

  return (
    <Container size="page" className="py-8">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} />}
      {stack && (
        <div className="space-y-6">
          <StackHeaderSection stack={stack} latest={latest} lastHealthy={lastHealthy} image={image} hostnames={hostnames} />
          <StackTabs stackId={stack.id} active={tab} showLogs={showLogs} deployCount={deploys?.length} />
          {tab === "overview" && <ServicesSection stackId={stack.id} />}
          {tab === "overview" && <RequestsSection stack={stack} />}
          {tab === "deploys" && <DeployHistorySection deploys={deploys} lastHealthy={lastHealthy} isLoading={deploysPending} error={deploysError} />}
          {tab === "logs" && <StackLogsSection stackId={stack.id} service={service} />}
          {tab === "settings" && (
            <SettingsShell section={section} nav={<StackNav stackId={stack.id} active={section} showBranches={showBranches} showExposures={showExposures} />}>
              {section === "environment" && <StackEnvironmentSection stack={stack} image={image} />}
              {section === "alerts" && <StackAlertsSection stack={stack} />}
              {section === "exposures" && <ServiceHostnameSection containers={services ?? []} />}
              {section === "branches" && <StackBranchDeploySection stack={stack} />}
              {section === "danger" && <StackDangerZoneSection stack={stack} projectPath={projectPath} hostnames={hostnames} />}
            </SettingsShell>
          )}
        </div>
      )}
    </Container>
  );
};
