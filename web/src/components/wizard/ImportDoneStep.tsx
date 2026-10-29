import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { GatewayAdoptionItem } from "@/components/wizard/GatewayAdoptionItem";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useAreaAccess } from "@/hooks/AccessHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import type { ImportResult } from "@/models/Machine";

interface ImportDoneStepProps {
  result: ImportResult;
}

export const ImportDoneStep = ({ result }: ImportDoneStepProps) => {
  const can = useAreaAccess();
  const wsPath = useWorkspacePath();
  return (
    <div className="space-y-4">
      <p className="text-sm">
        Imported {result.stacks.length} {result.stacks.length === 1 ? "stack" : "stacks"} as unmanaged.
      </p>
      {result.gateways.map((g) => (
        <GatewayAdoptionItem key={g.name} adoption={g} />
      ))}
      {(can?.("topology") ?? false) && (
        <WizardFooter>
          <Button asChild>
            <Link to={wsPath("/topology")}>View on the canvas</Link>
          </Button>
        </WizardFooter>
      )}
    </div>
  );
};
