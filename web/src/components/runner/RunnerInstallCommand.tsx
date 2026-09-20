import { Check, Copy } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";

interface RunnerInstallCommandProps {
  command: string;
}

// wrap+break-all (not just overflow-x-auto) so the command never forces horizontal scroll at 320px.
export const RunnerInstallCommand = ({ command }: RunnerInstallCommandProps) => {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard may be unavailable; the block stays visible for manual copy.
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium">Install command</p>
        <Button type="button" variant="outline" size="sm" onClick={copy}>
          {copied && <Check className="size-4" />}
          {!copied && <Copy className="size-4" />}
          {copied ? "Copied" : "Copy"}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded-md border border-border bg-muted p-3 font-mono text-xs break-all whitespace-pre-wrap">
        {command}
      </pre>
      <p className="text-xs text-muted-foreground">
        The binary is downloaded from this instance; the secret is shared by every runner. Keep it
        running with your service manager of choice.
      </p>
    </div>
  );
};
