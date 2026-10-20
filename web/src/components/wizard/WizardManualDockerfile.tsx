import { useState } from "react";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

// A scan that found nothing to deploy still goes on, from a Dockerfile path the person names.
export const WizardManualDockerfile = ({ onContinue }: { onContinue: (path: string) => void }) => {
  const [path, setPath] = useState("Dockerfile");

  return (
    <EmptyState
      title="Nothing to deploy found"
      message="No Dockerfile or compose file found. Enter a Dockerfile path to continue."
      size="compact"
      action={
        <div className="flex flex-wrap items-center justify-center gap-2">
          <Input value={path} onChange={(e) => setPath(e.target.value)} placeholder="Dockerfile" aria-label="Dockerfile path" className="h-8 w-40" />
          <Button size="sm" onClick={() => onContinue(path.trim())} disabled={!path.trim()}>
            Continue manually
          </Button>
        </div>
      }
    />
  );
};
