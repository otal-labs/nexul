import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { AccountStatus } from "@/models/Team";

interface AccountStatusLabelProps {
  status: AccountStatus;
}

// Only an account that can't sign in says so; an active one needs no word, its presence already reads beside it.
export const AccountStatusLabel = ({ status }: AccountStatusLabelProps) => (
  <>
    {status === "disabled" && <SettingsStatus tone="warning">Disabled</SettingsStatus>}
    {status === "removed" && <SettingsStatus tone="destructive">Removed</SettingsStatus>}
  </>
);
