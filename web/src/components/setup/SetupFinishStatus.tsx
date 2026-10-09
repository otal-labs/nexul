import { Loader2 } from "lucide-react";

import { errorMessage } from "@/api/client";
import { Button } from "@/components/ui/button";
import { useElapsedSeconds } from "@/hooks/useElapsedSeconds";
import type { useSetInstanceUrl } from "@/hooks/SetupHooks";

interface SetupFinishStatusProps {
  finish: ReturnType<typeof useSetInstanceUrl>;
}

// The shared end of every domain path: the server checks the address answers as Nexul over HTTPS, then stores it.
export const SetupFinishStatus = ({ finish }: SetupFinishStatusProps) => {
  const elapsed = useElapsedSeconds(finish.submittedAt, finish.isPending);
  const url = finish.variables?.url ?? "";
  const patient = (finish.variables?.attempts ?? 1) > 1;

  return (
    <div className="space-y-4">
      {finish.isPending && (
        <p role="status" className="flex items-start gap-2 text-sm">
          <Loader2 className="mt-0.5 size-4 shrink-0 animate-spin motion-reduce:animate-none" aria-hidden />
          <span className="min-w-0">
            Checking that <span className="break-all font-mono text-xs">{url}</span> answers as Nexul over HTTPS…
            <span className="block text-xs text-muted-foreground tabular-nums">
              {patient && "Let's Encrypt can take a couple of minutes to issue the certificate. "}
              {elapsed}s
            </span>
          </span>
        </p>
      )}
      {finish.isError && (
        <div className="space-y-3">
          <p role="alert" className="text-sm text-destructive">
            {url} didn&apos;t answer as this Nexul: {errorMessage(finish.error)}
          </p>
          <Button variant="outline" onClick={() => finish.variables && finish.mutate(finish.variables)}>
            Try again
          </Button>
        </div>
      )}
      {finish.isSuccess && <p className="text-sm text-muted-foreground">Saved {url} as this instance's address.</p>}
    </div>
  );
};
