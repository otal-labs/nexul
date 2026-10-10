import { Fact } from "@/components/Fact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";

interface GitHubAppKeyFactProps {
  set: boolean;
}

// How background work on attached repositories reads GitHub: as the App, or without a key as the connected account.
export const GitHubAppKeyFact = ({ set }: GitHubAppKeyFactProps) => (
  <div className="sm:col-span-3">
    <Fact label="Private key">
      {set && <SettingsStatus tone="success" detail="deploys read attached repositories as the App">Set</SettingsStatus>}
      {!set && (
        <SettingsStatus tone="warning" detail="deploys read attached repositories as the connected account">
          Not set
        </SettingsStatus>
      )}
    </Fact>
  </div>
);
