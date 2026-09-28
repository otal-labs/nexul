import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ComputersSection } from "@/components/settings/ComputersSection";
import { PairingDefaultsSection } from "@/components/settings/PairingDefaultsSection";

// Computers comes first so /settings?section=pairing&setup=<id> (no ?tab=) opens the setup summary it names.
export const PairingPanel = () => (
  <PageTabs
    label="T3 pairing settings"
    tabs={[
      { value: "computers", label: "Computers" },
      { value: "defaults", label: "Defaults" },
    ]}
  >
    <PageTabsContent value="computers">
      <ComputersSection />
    </PageTabsContent>
    <PageTabsContent value="defaults">
      <PairingDefaultsSection />
    </PageTabsContent>
  </PageTabs>
);
