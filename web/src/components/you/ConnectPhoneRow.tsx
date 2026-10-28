import { useState } from "react";
import { QrCode } from "lucide-react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { SettingsRow } from "@/components/settings/SettingsRow";
import { Button } from "@/components/ui/button";
import { IssuedConnectCode } from "@/components/you/IssuedConnectCode";
import { PhoneConnected } from "@/components/you/PhoneConnected";
import { useConnectCode } from "@/hooks/AuthHooks";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";

// The code opens under the row, across the card, so the row keeps its own height and the QR has room beside its text.
export const ConnectPhoneRow = () => {
  // A phone that signed in after this row appeared is the one this row's code let in; it stays confirmed until the page is left.
  const [mountedAt] = useState(() => Date.now());
  const connected = useDeviceArrivalStore((s) => s.arrivals.filter((a) => a.at >= mountedAt).at(-1));
  const { mutate, data, error, isPending } = useConnectCode();
  const generate = () => mutate();

  return (
    <div>
      <SettingsRow label="Phone app" description="Scan the code with the Nexul phone app. The code works once, for two minutes." className="last:pb-4">
        {!connected && !data && (
          <Button variant="outline" size="sm" onClick={generate} loading={isPending}>
            <QrCode className="size-4" aria-hidden />
            {error ? "Try again" : "Generate code"}
          </Button>
        )}
      </SettingsRow>
      {connected && (
        <div className="pb-4">
          <PhoneConnected label={connected.label} />
        </div>
      )}
      {!connected && error && (
        <div className="pb-4">
          <ErrorDisplay error={error} />
        </div>
      )}
      {!connected && data && (
        <div className="pb-4">
          <IssuedConnectCode code={data} onNewCode={generate} generating={isPending} hint={false} />
        </div>
      )}
    </div>
  );
};
