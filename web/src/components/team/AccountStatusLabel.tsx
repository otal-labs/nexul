import { SettingsStatus } from "@/components/settings/SettingsStatus";
import type { AccountStatus } from "@/models/Team";

interface AccountStatusLabelProps {
  status: AccountStatus;
}

// Only an account that can't sign in says so, Disabled in plain words (its muted name already marks it); an active one needs no word.
export const AccountStatusLabel = ({ status }: AccountStatusLabelProps) => (
  <>
    {status === "disabled" && <span className="shrink-0 text-xs text-muted-foreground">Disabled</span>}
    {status === "removed" && <SettingsStatus tone="destructive">Removed</SettingsStatus>}
  </>
);
