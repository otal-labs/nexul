import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { McpConfigSection } from "@/components/settings/McpConfigSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  del: vi.fn(),
  errorMessage: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, patch: mocks.patch, delete: mocks.del },
  errorMessage: mocks.errorMessage,
}));

const settings = {
  instance_url: "https://deploy.example.com",
  settings_version: 1,
  oauth_callback: "https://deploy.example.com/auth/callback",
  mention_chip_template: "",
  mcp_url: "https://deploy.example.com/mcp",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <McpConfigSection />
    </QueryClientProvider>,
  );
};

describe("McpConfigSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.errorMessage.mockClear();
    mocks.get.mockResolvedValue({ data: settings });
  });

  it("renders a snippet per provider with a placeholder token", async () => {
    renderSection();

    expect(await screen.findByText("Claude Code")).toBeInTheDocument();
    expect(screen.getByText("OpenCode")).toBeInTheDocument();
    const claudeSnippet = screen.getByText("Claude Code").closest("div")?.parentElement?.querySelector("pre");
    expect(claudeSnippet?.textContent).toContain(settings.mcp_url);
    expect(claudeSnippet?.textContent).toContain("<YOUR_PERSONAL_ACCESS_TOKEN>");
  });

  it("fills the snippets with a pasted token", async () => {
    const user = userEvent.setup();
    renderSection();

    const input = await screen.findByLabelText(/your personal access token/i);
    await user.type(input, "dep_abc123");

    const blocks = document.querySelectorAll("pre");
    expect(blocks.length).toBeGreaterThan(0);
    blocks.forEach((block) => {
      expect(block.textContent).toContain("dep_abc123");
      expect(block.textContent).not.toContain("<YOUR_PERSONAL_ACCESS_TOKEN>");
    });
  });

  it("prompts to set an instance URL when the MCP URL is unavailable", async () => {
    mocks.get.mockResolvedValue({ data: { ...settings, mcp_url: undefined } });
    renderSection();

    expect(await screen.findByText(/set an instance url in settings first/i)).toBeInTheDocument();
  });
});
