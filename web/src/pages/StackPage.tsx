import { useParams, useSearchParams } from "react-router";

import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ServiceHostnameSection } from "@/components/dns/ServiceHostnameSection";
import { DeployHistorySection } from "@/components/service/DeployHistorySection";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { ContainersTable } from "@/components/stack/ContainersTable";
import { StackBranchDeploySection } from "@/components/stack/StackBranchDeploySection";
import { StackDangerZoneSection } from "@/components/stack/StackDangerZoneSection";
import { StackDeployActions } from "@/components/stack/StackDeployActions";
import { StackHeaderSection } from "@/components/stack/StackHeaderSection";
import { DEFAULT_STACK_SECTION, isStackSection, StackNav, type StackSection } from "@/components/stack/StackNav";
import { useFetchExposures } from "@/hooks/DnsHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useFetchStack, useFetchStackDeploys, useFetchStackServices } from "@/hooks/StackHooks";
import { latestDeploy, type Container as StackContainer } from "@/models/Stack";
import { projectSettingsPath, projectTokenById } from "@/models/Project";

// A container's declared image, or the image observed running once the runner reports one.
const imageOf = (c: StackContainer | undefined): string | undefined => c?.image || c?.declared.image;

// Same section-per-view shape as the settings pages: ?section= drives the card, StackNav lists sections.
export const StackPage = () => {
  const { stackId = "" } = useParams();
  const { data: stack, isPending, error } = useFetchStack(stackId);
  const { data: projects = [] } = useFetchProjects();
  const { data: services, isPending: servicesPending, error: servicesError } = useFetchStackServices(stackId);
  const { data: deploys, isPending: deploysPending } = useFetchStackDeploys(stackId);
  const { data: exposures } = useFetchExposures();

  const [searchParams] = useSearchParams();
  const rawSection = searchParams.get("section");
  const requested: StackSection = isStackSection(rawSection) ? rawSection : DEFAULT_STACK_SECTION;
  const showBranches = !!stack && !stack.derived_from;
  const section = requested === "branches" && !showBranches ? DEFAULT_STACK_SECTION : requested;

  const latest = latestDeploy(deploys);
  const lastHealthy = deploys?.find((d) => d.status === "healthy");
  const canRollback = lastHealthy != null;
  const containerIds = new Set((services ?? []).map((c) => c.id));
  const hostnames = (exposures ?? [])
    .filter((e) => !!e.service_id && containerIds.has(e.service_id))
    .map((e) => e.hostname);
  const projectPath = stack ? projectSettingsPath(projectTokenById(projects, stack.project_id)) : "";
  const image = latest?.image || imageOf(services?.[0]);

  return (
    <Container className="mx-auto max-w-5xl py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {stack && (
        <>
          <StackHeaderSection
            stack={stack}
            projectPath={projectPath}
            latest={latest}
            image={image}
            hostnames={hostnames}
          />
          <div className="mt-6 flex flex-col gap-6 md:flex-row md:items-start md:gap-8">
            <StackNav active={section} showBranches={showBranches} />
            <div className="min-w-0 flex-1 space-y-6">
              {section === "overview" && (
                <>
                  <StackDeployActions stack={stack} lastHealthy={lastHealthy} canRollback={canRollback} image={image} />
                  <SettingsCard
                    id="services"
                    title="Services"
                    description="Containers this stack declares, with what the runner last observed for each."
                  >
                    {servicesPending && <LoadingDisplay />}
                    {servicesError && <ErrorDisplay error={servicesError} title="Could not load services" />}
                    {services && <ContainersTable containers={services} />}
                  </SettingsCard>
                </>
              )}
              {section === "exposures" && <ServiceHostnameSection containers={services ?? []} />}
              {section === "branches" && <StackBranchDeploySection stack={stack} />}
              {section === "history" && <DeployHistorySection deploys={deploys} isLoading={deploysPending} />}
              {section === "danger" && (
                <StackDangerZoneSection stack={stack} projectPath={projectPath} hostnames={hostnames} />
              )}
            </div>
          </div>
        </>
      )}
    </Container>
  );
};
