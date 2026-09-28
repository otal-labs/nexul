import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { AppearanceSection } from "@/components/settings/AppearanceSection";
import { ComputersSection } from "@/components/settings/ComputersSection";
import { PairingDefaultsSection } from "@/components/settings/PairingDefaultsSection";
import { ProfileSection } from "@/components/you/ProfileSection";
import { SecurityPanel } from "@/components/you/SecurityPanel";
import type { YourSettingsSection } from "@/components/you/YourSettingsNav";

export const YourSettingsContent = ({ section }: { section: YourSettingsSection }) => (
  <>
    {section === "profile" && <ProfileSection />}
    {section === "appearance" && <AppearanceSection />}
    {section === "security" && <SecurityPanel />}
    {section === "pairing" && (
      <PageTabs
        label="T3 pairing"
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
    )}
  </>
);
