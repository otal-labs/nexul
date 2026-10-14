import { OpenEnrollmentsSection } from "@/components/runner/OpenEnrollmentsSection";
import { useOpenEnrollments } from "@/hooks/RunnerHooks";

// Runner codes nobody has used yet, each with Revoke; the viewer sees the ones they may revoke.
export const OpenRunnerCodes = () => {
  const { data: codes } = useOpenEnrollments(false);
  return <OpenEnrollmentsSection heading="Unused runner codes" codes={codes} labelOf={(code) => code.name} />;
};
