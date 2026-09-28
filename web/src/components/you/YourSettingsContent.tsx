import { AppearanceSection } from "@/components/settings/AppearanceSection";
import { PairingPanel } from "@/components/settings/PairingPanel";
import { ProfileSection } from "@/components/you/ProfileSection";
import { SecurityPanel } from "@/components/you/SecurityPanel";
import type { YourSettingsSection } from "@/components/you/YourSettingsNav";

interface YourSettingsContentProps {
  section: YourSettingsSection;
}

export const YourSettingsContent = ({ section }: YourSettingsContentProps) => (
  <>
    {section === "profile" && <ProfileSection />}
    {section === "appearance" && <AppearanceSection />}
    {section === "security" && <SecurityPanel />}
    {section === "pairing" && <PairingPanel />}
  </>
);
