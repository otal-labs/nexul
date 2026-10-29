import { useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { Button } from "@/components/ui/button";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { ImportAdoptStep } from "@/components/wizard/ImportAdoptStep";
import { ImportDoneStep } from "@/components/wizard/ImportDoneStep";
import { ImportRail, type ImportStage } from "@/components/wizard/ImportRail";
import { ImportSourceStep } from "@/components/wizard/ImportSourceStep";
import { WizardShell } from "@/components/wizard/WizardShell";
import { WizardStepHeader } from "@/components/wizard/WizardStepHeader";
import { useCanAddMachines } from "@/hooks/AccessHooks";
import { useDiscoverMachine, useFetchMachines, useImportMachine } from "@/hooks/MachineHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useImportSelection } from "@/hooks/useImportSelection";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { GroupedDiscovery, ImportResult } from "@/models/Machine";

const COPY: Record<ImportStage, { title: string; lead: string }> = {
  source: { title: "Import from a machine", lead: "Adopt what's already running as unmanaged stacks." },
  adopt: { title: "What to adopt", lead: "Everything is ticked; untick what Nexul should leave alone." },
  done: { title: "Imported", lead: "Adopted as unmanaged stacks. Attach a repository to one to manage it." },
};

// Pick a machine and a project, discover what runs there, adopt the selection as unmanaged stacks.
export const ImportWizard = () => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const canAddMachines = useCanAddMachines();
  const [searchParams] = useSearchParams();
  const [machineId, setMachineId] = useState(() => searchParams.get("machine") ?? "");
  const [projectId, setProjectId] = useState("");
  const [report, setReport] = useState<GroupedDiscovery | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const { data: machines, isPending, error } = useFetchMachines();
  const { data: projects } = useFetchProjects();
  const discover = useDiscoverMachine();
  const doImport = useImportMachine();
  const pick = useImportSelection();
  const stage: ImportStage = (result && "done") || (report && "adopt") || "source";

  const runDiscover = async () => {
    try {
      const found = await discover.mutateAsync(machineId);
      setReport(found);
      pick.seed(found);
    } catch {
      // Errors surface through the hook's toast; discovery is retry-safe.
    }
  };

  const runImport = async () => {
    try {
      setResult(await doImport.mutateAsync({ machineId, input: pick.toRequest(projectId) }));
    } catch {
      // Errors surface through the hook's toast; import is retry-safe (it matches by name, never duplicates).
    }
  };

  return (
    <WizardShell
      rail={<ImportRail stage={stage} />}
      exit={
        <Button variant="ghost" size="sm" className="-mr-2 h-7 px-2 text-muted-foreground" onClick={() => navigate(wsPath(canAddMachines ? "/runners" : "/"))}>
          Exit
        </Button>
      }
    >
      <WizardStepHeader title={COPY[stage].title} lead={COPY[stage].lead} />
      <div className="mt-8">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} title="Couldn't load machines." />}
        {machines && stage === "source" && (
          <ImportSourceStep
            machines={machines}
            projects={projects}
            machineId={machineId}
            projectId={projectId}
            onMachine={setMachineId}
            onProject={setProjectId}
            onDiscover={() => void runDiscover()}
            discovering={discover.isPending}
          />
        )}
        {report && stage === "adopt" && (
          <ImportAdoptStep report={report} pick={pick} onBack={() => setReport(null)} onImport={() => void runImport()} importing={doImport.isPending} />
        )}
        {result && <ImportDoneStep result={result} />}
      </div>
    </WizardShell>
  );
};
