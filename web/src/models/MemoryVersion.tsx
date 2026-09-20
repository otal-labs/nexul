// Mirrors internal/memories/model.go's MemoryVersion: one append-only row per save, revert included.
export interface MemoryVersion {
  id: string;
  memory_id: string;
  version: number;
  title: string;
  when_to_use: string;
  body: string;
  always_included: boolean;
  author_id: string;
  // author_via is "mcp" when the save came from an MCP tool call, empty for the browser.
  author_via: string;
  created_at: string;
}
