import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { NewAutomationPanel } from "@/components/automation/NewAutomationPanel";

const mocks = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post, get: mocks.get },
  errorMessage: () => "error",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderPanel = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <NewAutomationPanel />
    </QueryClientProvider>,
  );
};

describe("NewAutomationPanel", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.get.mockReset();
    mocks.get.mockImplementation((url: string) => {
      if (url === "/api/integrations/scopes") {
        return Promise.resolve({
          data: {
            scopes: [
              { value: "docs:read", label: "Read docs", domain: "docs", action: "read" },
              { value: "tickets:read", label: "Read tickets", domain: "tickets", action: "read" },
              { value: "tickets:write", label: "Create and update tickets", domain: "tickets", action: "write" },
              { value: "events:read", label: "Read events", domain: "events", action: "read" },
            ],
          },
        });
      }
      return Promise.resolve({ data: { instance_url: "https://demo.nexul.com" } });
    });
  });

  it("opens the create form, then reveals the minted token and SDK commands on success", async () => {
    mocks.post.mockResolvedValue({
      data: {
        automation: { id: "a1", name: "Slack notifier" },
        token: "dep_secret_abc123",
      },
    });
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole("button", { name: "New automation" }));
    await user.type(screen.getByLabelText("Name"), "Slack notifier");
    await user.click(await screen.findByRole("checkbox", { name: /Create and update tickets/ }));
    await user.click(screen.getByRole("checkbox", { name: /Read events/ }));
    await user.click(screen.getByRole("button", { name: "Create automation" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations", {
      name: "Slack notifier",
      scopes: ["tickets:write", "events:read"],
    });
    expect(await screen.findByText("Slack notifier created")).toBeInTheDocument();
    expect(screen.getByText("dep_secret_abc123")).toBeInTheDocument();
    expect(screen.getByText(/npx @nexul\/sdk init/)).toBeInTheDocument();
  });

  it("unticking a scope removes it from the submitted set", async () => {
    mocks.post.mockResolvedValue({ data: { automation: { id: "a1", name: "x" }, token: "t" } });
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole("button", { name: "New automation" }));
    await user.type(screen.getByLabelText("Name"), "x");
    const docs = await screen.findByRole("checkbox", { name: /Read docs/ });
    await user.click(docs);
    await user.click(screen.getByRole("checkbox", { name: /Read tickets/ }));
    await user.click(docs);
    await user.click(screen.getByRole("button", { name: "Create automation" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations", { name: "x", scopes: ["tickets:read"] });
  });

  it("requires a name and at least one scope before submitting", async () => {
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole("button", { name: "New automation" }));
    await user.click(screen.getByRole("button", { name: "Create automation" }));

    expect(await screen.findByText("Name is required")).toBeInTheDocument();
    expect(screen.getByText("Pick at least one scope")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
