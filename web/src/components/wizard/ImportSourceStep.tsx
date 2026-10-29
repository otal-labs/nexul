import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { Machine } from "@/models/Machine";
import type { Project } from "@/models/Project";
import { NEW_PROJECT_PATH } from "@/models/Project";

interface ImportSourceStepProps {
  machines: Machine[];
  projects: Project[] | undefined;
  machineId: string;
  projectId: string;
  onMachine: (id: string) => void;
  onProject: (id: string) => void;
  onDiscover: () => void;
  discovering: boolean;
}

// Where it comes from and where it goes: the machine to read and the project the adopted stacks join.
export const ImportSourceStep = ({ machines, projects, machineId, projectId, onMachine, onProject, onDiscover, discovering }: ImportSourceStepProps) => {
  const wsPath = useWorkspacePath();
  return (
    <div className="space-y-6">
      <div className="max-w-80 space-y-2">
        <label className="text-sm font-medium">Machine</label>
        <Select value={machineId} onValueChange={onMachine}>
          <SelectTrigger aria-label="Machine" className="w-full">
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
      <div className="max-w-80 space-y-2">
        <label className="text-sm font-medium">Project</label>
        <Select value={projectId} onValueChange={onProject}>
          <SelectTrigger aria-label="Project" className="w-full">
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
        <p className="text-xs text-muted-foreground">Adopted stacks belong to it.</p>
      </div>
      {projects && projects.length === 0 && (
        <p className="text-sm text-muted-foreground">
          Adopted stacks belong to a project.{" "}
          <Link to={wsPath(NEW_PROJECT_PATH)} className="text-foreground underline underline-offset-2">
            Create a project
          </Link>{" "}
          first.
        </p>
      )}
      <WizardFooter>
        <Button onClick={onDiscover} loading={discovering} disabled={!machineId || !projectId}>
          Discover
        </Button>
      </WizardFooter>
    </div>
  );
};
