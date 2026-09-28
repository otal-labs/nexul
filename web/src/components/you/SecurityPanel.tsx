import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ConnectionTokenSection } from "@/components/settings/ConnectionTokenSection";
import { PersonalAccessTokensSection } from "@/components/settings/PersonalAccessTokensSection";

// One tab for now, so PageTabs shows no tab row; Devices lands in front of it later.
export const SecurityPanel = () => (
  <PageTabs label="Security" tabs={[{ value: "tokens", label: "Tokens" }]}>
    <PageTabsContent value="tokens">
      <ConnectionTokenSection />
      <PersonalAccessTokensSection />
    </PageTabsContent>
  </PageTabs>
);
