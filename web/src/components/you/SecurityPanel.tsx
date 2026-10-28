import { PersonalAccessTokensSection } from "@/components/settings/PersonalAccessTokensSection";
import { ConnectDeviceCard } from "@/components/you/ConnectDeviceCard";
import { DevicesFeed } from "@/components/you/DevicesFeed";

// One page in the order of the job: who is signed in, what can act as you, then connecting a new device.
export const SecurityPanel = () => (
  <>
    <DevicesFeed />
    <PersonalAccessTokensSection />
    <ConnectDeviceCard />
  </>
);
