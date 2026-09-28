import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { PersonalAccessTokensSection } from "@/components/settings/PersonalAccessTokensSection";
import { ConnectDesktopCard } from "@/components/you/ConnectDesktopCard";
import { DevicesFeed } from "@/components/you/DevicesFeed";

export const SecurityPanel = () => (
  <PageTabs
    label="Security"
    tabs={[
      { value: "devices", label: "Devices" },
      { value: "tokens", label: "Tokens" },
    ]}
  >
    <PageTabsContent value="devices">
      {/* Full width until the Connect a phone card joins it in a two-column grid. */}
      <ConnectDesktopCard />
      <DevicesFeed />
    </PageTabsContent>
    <PageTabsContent value="tokens">
      <PersonalAccessTokensSection />
    </PageTabsContent>
  </PageTabs>
);
