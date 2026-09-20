import { Link, useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { Button } from "@/components/ui/button";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

// Terminal rung: a summary of what the wizard just built, a link to the stack it created (/stacks/<id>, the
// service page's replacement), and a way into the canvas to see it in context.
export const WizardDoneStep = () => {
  const navigate = useNavigate();
  const { name, machine, exposureHostname, stackId } = useProjectWizardStore(
    useShallow((s) => ({ name: s.name, machine: s.machine, exposureHostname: s.exposureHostname, stackId: s.stackId })),
  );
  const reset = useProjectWizardStore((s) => s.reset);

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm">
          {name} is deploying on {machine}.
        </p>
        {exposureHostname && (
          <p className="mt-1 font-mono text-xs text-muted-foreground">{exposureHostname}</p>
        )}
      </div>
      <div className="flex flex-wrap gap-3">
        {stackId && (
          <Button variant="outline" asChild onClick={reset}>
            <Link to={`/stacks/${stackId}`}>View stack</Link>
          </Button>
        )}
        <Button
          onClick={() => {
            reset();
            navigate("/topology");
          }}
        >
          View on the canvas
        </Button>
      </div>
    </div>
  );
};
