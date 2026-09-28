import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { PersonalAccessTokensSection } from "@/components/settings/PersonalAccessTokensSection";
import { ConnectDesktopCard } from "@/components/you/ConnectDesktopCard";
import { ConnectPhoneCard } from "@/components/you/ConnectPhoneCard";
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
      <div className="grid gap-6 lg:grid-cols-2 [&>*]:min-w-0">
        <ConnectPhoneCard />
        <ConnectDesktopCard />
      </div>
      <DevicesFeed />
    </PageTabsContent>
    <PageTabsContent value="tokens">
      <PersonalAccessTokensSection />
    </PageTabsContent>
  </PageTabs>
);
