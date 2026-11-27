import { ComputerPane } from "@/components/computers/ComputerPane";
import { ComputersList } from "@/components/computers/ComputersList";
import { DefaultsRunPane } from "@/components/computers/DefaultsRunPane";
import { ProjectRunPane } from "@/components/computers/ProjectRunPane";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { useListOrderComputers } from "@/hooks/PairingHooks";
import { DEFAULTS_ID, PROJECT_TARGET_PREFIX } from "@/models/Agent";

interface ComputersBodyProps {
  // The `computers` param: "" the first computer, a computer's id, "defaults", or "project:<id>".
  target: string;
}

// The list beside the open record; each column scrolls on its own from 768px, and the whole body scrolls below it.
export const ComputersBody = ({ target }: ComputersBodyProps) => {
  const { data: computers } = useListOrderComputers();
  const current = target === "" ? (computers?.[0]?.id ?? "") : target;
  const isProject = current.startsWith(PROJECT_TARGET_PREFIX);
  const isComputer = current !== "" && current !== DEFAULTS_ID && !isProject;

  return (
    <div className="grid min-h-0 flex-1 max-md:overflow-y-auto md:grid-cols-[18rem_minmax(0,1fr)]">
      <nav aria-label="Computers and run settings" className="scroll-edge border-b border-border py-2 md:overflow-y-auto md:border-r md:border-b-0">
        <ComputersList current={current} />
      </nav>
      <section aria-label="Details" className="scroll-edge p-4 sm:p-6 md:overflow-y-auto">
        {isComputer && <ComputerPane key={current} computerId={current} />}
        {current === DEFAULTS_ID && <DefaultsRunPane />}
        {isProject && <ProjectRunPane key={current} projectId={current.slice(PROJECT_TARGET_PREFIX.length)} />}
        {computers && computers.length === 0 && current === "" && <NoDataDisplay message="No computers yet. Add one to run plays on it." />}
      </section>
    </div>
  );
};
