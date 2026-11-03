import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useInstallTool } from "@/hooks/ComputerToolHooks";
import type { ToolFacts } from "@/models/ComputerFacts";

interface ToolFactLineProps {
  tool: ToolFacts;
  installing: boolean;
  onInstall: () => void;
}

const ToolFactLine = ({ tool, installing, onInstall }: ToolFactLineProps) => (
  <li className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
    <span className="font-mono text-foreground">{tool.name}</span>
    <SettingsStatus tone={tool.path ? "success" : "warning"} detail={tool.version && <span className="font-mono tabular-nums">{tool.version}</span>}>
      {tool.path ? "Installed" : "Not installed"}
    </SettingsStatus>
    {tool.path && <span className="min-w-0 font-mono break-all text-muted-foreground">{tool.path}</span>}
    {!tool.path && tool.install && tool.install.length > 0 && (
      <Button type="button" variant="outline" size="sm" className="h-7" loading={installing} onClick={onInstall}>
        Install {tool.name}
      </Button>
    )}
  </li>
);

interface ComputerToolFactsProps {
  computerId: string;
  tools: ToolFacts[];
}

// The tools Nexul runs on the computer, each installed with its version and path, or missing with a way to install it.
export const ComputerToolFacts = ({ computerId, tools }: ComputerToolFactsProps) => {
  const { install, installing } = useInstallTool(computerId);
  return (
    <ComputerFact label="Tools">
      <ul className="space-y-1">
        {tools.map((tool) => (
          <ToolFactLine key={tool.name} tool={tool} installing={installing === tool.name} onInstall={() => void install(tool.name)} />
        ))}
      </ul>
    </ComputerFact>
  );
};
