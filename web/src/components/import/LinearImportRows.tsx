import { Input } from "@/components/ui/input";
import { SettingsRow } from "@/components/settings/SettingsRow";

interface LinearImportRowsProps {
  token: string;
  team: string;
  onToken: (token: string) => void;
  onTeam: (team: string) => void;
}

// What a Linear import reads: a personal API key, and the team to read.
export const LinearImportRows = ({ token, team, onToken, onTeam }: LinearImportRowsProps) => (
  <>
    <SettingsRow
      label="API key"
      htmlFor="import-linear-key"
      description="A personal API key from Linear's security settings. It is deleted when the import ends."
    >
      <Input id="import-linear-key" type="password" autoComplete="off" value={token} onChange={(e) => onToken(e.target.value)} />
    </SettingsRow>
    <SettingsRow label="Team" htmlFor="import-linear-team" description="The team's key. Leave it empty when the API key reaches one team.">
      <Input id="import-linear-team" placeholder="ENG" value={team} onChange={(e) => onTeam(e.target.value)} />
    </SettingsRow>
  </>
);
