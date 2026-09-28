import { useState } from "react";

import { SettingsCard } from "@/components/settings/SettingsCard";
import { ConnectCodePanel } from "@/components/you/ConnectCodePanel";
import { PhoneConnected } from "@/components/you/PhoneConnected";
import { useDeviceArrivalStore } from "@/stores/deviceArrivalStore";

export const ConnectPhoneCard = () => {
  // A phone that signed in after this card appeared is the one this card's code let in; it stays confirmed until the page is left.
  const [mountedAt] = useState(() => Date.now());
  const connected = useDeviceArrivalStore((s) => s.arrivals.filter((a) => a.at >= mountedAt).at(-1));

  return (
    <SettingsCard id="connect-phone" title="Connect a phone" description="Open the Nexul app on your phone and scan this code.">
      {connected && <PhoneConnected label={connected.label} />}
      {!connected && <ConnectCodePanel />}
    </SettingsCard>
  );
};
