import { useState } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { McpSnippetCard } from "@/components/settings/McpSnippetCard";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Input } from "@/components/ui/input";
import { useFetchSettings } from "@/hooks/AuthHooks";

const PLACEHOLDER_TOKEN = "<YOUR_PERSONAL_ACCESS_TOKEN>";

// Per-provider MCP snippet; reuses the PAT mint flow — the token here is pasted, not minted.
const PROVIDER_SNIPPETS = [
  {
    id: "claude",
    label: "Claude Code",
    filename: ".mcp.json",
    build: (url: string, token: string) =>
      JSON.stringify(
        { mcpServers: { nexul: { type: "http", url, headers: { Authorization: `Bearer ${token}` } } } },
        null,
        2,
      ),
  },
  {
    id: "opencode",
    label: "OpenCode",
    filename: "opencode.json",
    build: (url: string, token: string) =>
      JSON.stringify(
        { mcp: { nexul: { type: "remote", url, headers: { Authorization: `Bearer ${token}` }, enabled: true } } },
        null,
        2,
      ),
  },
];

export const McpConfigSection = () => {
  const { data: settings, isPending, error } = useFetchSettings();
  const [token, setToken] = useState("");
  const url = settings?.mcp_url ?? "";

  return (
    <SettingsCard
      id="pairing-mcp"
      title="MCP config"
      description="Give T3 Code — or any MCP-capable agent tool — access to this workspace. Mint a personal
        access token below (Personal access tokens), paste it in here, then copy the snippet for your
        provider."
    >
      <div className="space-y-4">
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {settings && !url && (
          <p className="text-sm text-muted-foreground">
            Set an instance URL in Settings first — the MCP URL derives from it.
          </p>
        )}
        {settings && url && (
          <div className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="mcp-pat" className="text-sm font-medium">
                Your personal access token
              </label>
              <Input
                id="mcp-pat"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="dep_… (paste a token minted below)"
                className="font-mono text-xs"
              />
            </div>
            <div className="space-y-3">
              {PROVIDER_SNIPPETS.map((provider) => (
                <McpSnippetCard
                  key={provider.id}
                  label={provider.label}
                  filename={provider.filename}
                  snippet={provider.build(url, token.trim() || PLACEHOLDER_TOKEN)}
                />
              ))}
            </div>
          </div>
        )}
      </div>
    </SettingsCard>
  );
};
