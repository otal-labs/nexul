import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRows } from "@/components/settings/SettingsRow";
import { ConnectDesktopRow } from "@/components/you/ConnectDesktopRow";
import { ConnectPhoneRow } from "@/components/you/ConnectPhoneRow";

// Done once per device, so it comes after the lists a person opens Security to check.
export const ConnectDeviceCard = () => (
  <SettingsCard id="connect-device" title="Connect a device" description="Sign the phone or desktop app in to this server.">
    <SettingsRows>
      <ConnectPhoneRow />
      <ConnectDesktopRow />
    </SettingsRows>
  </SettingsCard>
);
