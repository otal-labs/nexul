import { useParams } from "react-router";

import { Container } from "@/components/Container";
import { DeployHeaderSection } from "@/components/deploy/DeployHeaderSection";
import { DeployProgressSection } from "@/components/deploy/DeployProgressSection";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchDeploy } from "@/hooks/DeployHooks";

export const DeployPage = () => {
  const { deployId = "" } = useParams();
  const { data: deploy, isPending, error } = useFetchDeploy(deployId);

  return (
    <Container className="max-w-3xl py-8">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {deploy && <DeployHeaderSection deploy={deploy} />}
      {deploy && <DeployProgressSection deploy={deploy} />}
    </Container>
  );
};
