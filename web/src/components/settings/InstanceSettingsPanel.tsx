import { InstanceUrlSection } from "@/components/settings/InstanceUrlSection";
import { InstanceVersionSection } from "@/components/settings/InstanceVersionSection";
import type { InstanceSettings } from "@/models/User";

interface InstanceSettingsPanelProps {
  settings: InstanceSettings;
}

export const InstanceSettingsPanel = ({ settings }: InstanceSettingsPanelProps) => (
  <>
    <InstanceVersionSection />
    <InstanceUrlSection settings={settings} />
  </>
);
