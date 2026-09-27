import { CheckCircle2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useSetupPassStore } from "@/stores/setupPassStore";
import { handoffLink } from "@/models/Setup";

interface SetupHandoffProps {
  instanceUrl: string;
}

// A plain link, not a fetch: the domain is another origin, and the session and OAuth cookie must start there.
export const SetupHandoff = ({ instanceUrl }: SetupHandoffProps) => {
  const code = useSetupPassStore((s) => s.code);

  return (
    <div className="space-y-5">
      <p className="flex items-start gap-2 text-sm">
        <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success" aria-hidden />
        <span className="min-w-0">
          Nexul is live at <span className="break-all font-mono text-xs">{instanceUrl}</span>
        </span>
      </p>
      {!code && (
        <p className="text-sm text-muted-foreground">You will enter the setup code once more on the domain.</p>
      )}
      <Button asChild className="w-full">
        <a href={handoffLink(instanceUrl, code)}>Continue there</a>
      </Button>
    </div>
  );
};
