import { useState } from "react";
import { Link, useSearchParams } from "react-router";

import { Container } from "@/components/Container";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { PageHeader } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ImportContainerRow } from "@/components/wizard/ImportContainerRow";
import { ImportGatewayRow } from "@/components/wizard/ImportGatewayRow";
import { ImportStackGroupSection } from "@/components/wizard/ImportStackGroupSection";
import { useDiscoverMachine, useFetchMachines, useImportMachine } from "@/hooks/MachineHooks";
import { useFetchProjects } from "@/hooks/ProjectHooks";
import { useImportSelection } from "@/hooks/useImportSelection";
import type { GroupedDiscovery, ImportResult } from "@/models/Machine";

// Door 3 (spec §1/§8): pick a machine, discover what's already running on it, adopt the selection as unmanaged
// stacks. A single self-contained page, not a rail — there's no "later step" that depends on an earlier one
// the way the project wizard's steps do.
export const ProjectWizardImportPage = () => {
  const [searchParams] = useSearchParams();
  const [machineId, setMachineId] = useState(() => searchParams.get("machine") ?? "");
  const [projectId, setProjectId] = useState("");
  const [report, setReport] = useState<GroupedDiscovery | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const { data: machines, isPending: machinesPending, error: machinesError } = useFetchMachines();
  const { data: projects } = useFetchProjects();
  const discover = useDiscoverMachine();
  const doImport = useImportMachine();
  const { selection, seed, toggleContainer, toggleStandalone, toggleGateway, toRequest } = useImportSelection();

  const runDiscover = async () => {
    if (!machineId) return;
    try {
      const found = await discover.mutateAsync(machineId);
      setReport(found);
      seed(found);
    } catch {
      // Errors surface through the hook's toast; discovery is retry-safe.
    }
  };

  const runImport = async () => {
    if (!machineId || !projectId) return;
    try {
      setResult(await doImport.mutateAsync({ machineId, input: toRequest(projectId) }));
    } catch {
      // Errors surface through the hook's toast; import is retry-safe (it matches by name, never duplicates).
    }
  };

  const nothingFound =
    report && report.stacks.length === 0 && report.standalone.length === 0 && report.gateways.length === 0;

  return (
    <Container className="max-w-2xl py-10">
      <PageHeader title="Import from this machine" subtitle="Adopt what's already running as unmanaged stacks." />
      <div className="mt-6 space-y-6">
        {machinesPending && <LoadingDisplay />}
        {machinesError && <ErrorDisplay error={machinesError} title="Could not load machines" />}
        {machines && !report && (
          <div className="flex flex-wrap items-end gap-3">
            <div className="space-y-2">
              <label className="text-sm font-medium">Machine</label>
              <Select value={machineId} onValueChange={setMachineId}>
                <SelectTrigger aria-label="Machine">
                  <SelectValue placeholder="Choose a machine…" />
                </SelectTrigger>
                <SelectContent>
                  {machines.map((machine) => (
                    <SelectItem key={machine.id} value={machine.id}>
                      {machine.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Button onClick={() => void runDiscover()} disabled={!machineId || discover.isPending}>
              {discover.isPending ? "Scanning…" : "Discover"}
            </Button>
          </div>
        )}
        {nothingFound && <NoDataDisplay message="Nothing found on this machine." />}
        {report && !result && !nothingFound && (
          <div className="space-y-6">
            <div className="space-y-4 divide-y divide-border">
              {report.stacks.map((group) => (
                <ImportStackGroupSection
                  key={group.project}
                  group={group}
                  selected={selection.stacks[group.project] ?? new Set()}
                  onToggleContainer={(name) => toggleContainer(group.project, name)}
                />
              ))}
              {report.standalone.map((c) => (
                <ImportContainerRow
                  key={c.name}
                  container={c}
                  checked={selection.standalone.has(c.name)}
                  onToggle={() => toggleStandalone(c.name)}
                />
              ))}
              {report.gateways.map((c) => (
                <ImportGatewayRow
                  key={c.name}
                  container={c}
                  checked={selection.gateways.has(c.name)}
                  onToggle={() => toggleGateway(c.name)}
                />
              ))}
            </div>
            <div className="flex flex-wrap items-end gap-3">
              <div className="space-y-2">
                <label className="text-sm font-medium">Project</label>
                <Select value={projectId} onValueChange={setProjectId}>
                  <SelectTrigger aria-label="Project">
                    <SelectValue placeholder="Choose a project…" />
                  </SelectTrigger>
                  <SelectContent>
                    {(projects ?? []).map((project) => (
                      <SelectItem key={project.id} value={project.id}>
                        {project.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button onClick={() => void runImport()} disabled={!projectId || doImport.isPending}>
                {doImport.isPending ? "Importing…" : "Import"}
              </Button>
            </div>
          </div>
        )}
        {result && (
          <div className="space-y-4">
            <p className="text-sm">Imported {result.stacks.length} stack(s) as unmanaged.</p>
            {result.gateways.map((g) => (
              <div key={g.name} className="space-y-1 text-sm">
                <p className="font-medium">{g.name}</p>
                {g.error && <p className="text-xs text-destructive">Not adopted as a gateway: {g.error}</p>}
                {g.gateway_id && g.exposed.length === 0 && g.unmatched.length === 0 && (
                  <p className="text-xs text-muted-foreground">Adopted as a gateway; no hostnames routed through it yet.</p>
                )}
                {g.exposed.length > 0 && (
                  <ul aria-label={`Hostnames now on the canvas via ${g.name}`} className="space-y-0.5 font-mono text-xs text-muted-foreground">
                    {g.exposed.map((host) => (
                      <li key={host}>{host}</li>
                    ))}
                  </ul>
                )}
                {g.unmatched.length > 0 && (
                  <p className="text-xs text-muted-foreground">
                    Not linked, their target is not a container here: {g.unmatched.join(", ")}
                  </p>
                )}
              </div>
            ))}
            <Button asChild>
              <Link to="/topology">View on the canvas</Link>
            </Button>
          </div>
        )}
      </div>
    </Container>
  );
};
