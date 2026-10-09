import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DeployHeaderSection } from "@/components/deploy/DeployHeaderSection";
import { DeployProgressSection } from "@/components/deploy/DeployProgressSection";
import { DetailErrorDisplay } from "@/components/DetailErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDeploy } from "@/hooks/DeployHooks";

export const DeployPage = () => {
  const { deployId = "" } = useParams();
  const { data: deploy, isPending, error } = useFetchDeploy(deployId);

  return (
    <Container size="page" className="@container py-8">
      {isPending && <LoadingDisplay />}
      {error && <DetailErrorDisplay error={error} />}
      {deploy && <DeployHeaderSection deploy={deploy} />}
      {deploy && <DeployProgressSection deploy={deploy} />}
    </Container>
  );
};
