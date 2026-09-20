import { ConnectorAppConfigSection } from "@/components/settings/ConnectorAppConfigSection";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";

interface ConnectorsSettingsPanelProps {
  isInstanceAdmin: boolean;
}

export const ConnectorsSettingsPanel = ({ isInstanceAdmin }: ConnectorsSettingsPanelProps) => (
  <>
    {isInstanceAdmin && <ConnectorAppConfigSection />}
    <ConnectorsSection />
  </>
);
