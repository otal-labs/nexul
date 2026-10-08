import { QrCode } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { Button } from "@/components/ui/button";
import { IssuedConnectCode } from "@/components/you/IssuedConnectCode";
import { QrFrame } from "@/components/you/QrFrame";
import { useConnectCode } from "@/hooks/AuthHooks";

export const ConnectCodePanel = () => {
  const { mutate, data, error, isPending } = useConnectCode();
  const generate = () => mutate();

  return (
    <>
      {error && <ErrorDisplay error={error} />}
      {error && (
        <Button variant="outline" size="sm" className="mt-3" onClick={generate} loading={isPending}>
          Try again
        </Button>
      )}
      {!error && !data && (
        <QrFrame
          dimmed
          action={
            <Button size="sm" className="absolute inset-0 m-auto w-fit" onClick={generate} loading={isPending}>
              <QrCode className="size-4" aria-hidden />
              Generate code
            </Button>
          }
        >
          <p className="text-muted-foreground">The code works once, for two minutes.</p>
        </QrFrame>
      )}
      {data && <IssuedConnectCode code={data} onNewCode={generate} generating={isPending} />}
    </>
  );
};
