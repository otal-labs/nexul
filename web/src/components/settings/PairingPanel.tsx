import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { ComputersSection } from "@/components/settings/ComputersSection";
import { PairingDefaultsSection } from "@/components/settings/PairingDefaultsSection";
import { PairingProjectsSection } from "@/components/settings/PairingProjectsSection";

// Computers comes first so /settings/pairing?setup=<id> (no tab segment) opens the setup summary it names.
export const PairingPanel = () => (
  <PageTabs
    label="T3 pairing settings"
    tabs={[
      { value: "computers", label: "Computers" },
      { value: "projects", label: "Projects" },
      { value: "defaults", label: "Defaults" },
    ]}
  >
    <PageTabsContent value="computers">
      <ComputersSection />
    </PageTabsContent>
    <PageTabsContent value="projects">
      <PairingProjectsSection />
    </PageTabsContent>
    <PageTabsContent value="defaults">
      <PairingDefaultsSection />
    </PageTabsContent>
  </PageTabs>
);
