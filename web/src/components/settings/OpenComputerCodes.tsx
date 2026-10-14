import { OpenEnrollmentsSection } from "@/components/runner/OpenEnrollmentsSection";
import { useListComputers } from "@/hooks/PairingHooks";
import { useOpenEnrollments } from "@/hooks/RunnerHooks";

// The codes that add the viewer's own computers and nobody has used yet, each with Revoke.
export const OpenComputerCodes = () => {
  const { data: codes } = useOpenEnrollments(true);
  const { data: computers } = useListComputers();
  const nameOf = (computerId: string | undefined) => computers?.find((c) => c.id === computerId)?.name ?? "A computer";
  return <OpenEnrollmentsSection heading="Unused computer codes" codes={codes} labelOf={(code) => nameOf(code.computer_id)} />;
};
