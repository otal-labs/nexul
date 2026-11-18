import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { SettingsRow } from "@/components/settings/SettingsRow";

interface NotionImportRowsProps {
  token: string;
  pages: string;
  onToken: (token: string) => void;
  onPages: (pages: string) => void;
}

// What a Notion import reads: the integration's secret, and the root pages to start from.
export const NotionImportRows = ({ token, pages, onToken, onPages }: NotionImportRowsProps) => (
  <>
    <SettingsRow
      label="Integration secret"
      htmlFor="import-notion-token"
      description="From an internal integration in Notion. It is deleted when the import ends."
    >
      <Input
        id="import-notion-token"
        type="password"
        autoComplete="off"
        value={token}
        onChange={(e) => onToken(e.target.value)}
      />
    </SettingsRow>
    <SettingsRow
      label="Root pages"
      htmlFor="import-notion-pages"
      description="Links to the pages to import, one per line; each becomes a folder. Leave it empty for every top page shared with the integration."
    >
      <Textarea
        id="import-notion-pages"
        rows={3}
        className="font-mono"
        value={pages}
        onChange={(e) => onPages(e.target.value)}
      />
    </SettingsRow>
  </>
);
