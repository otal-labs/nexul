import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { NewAutomationDialog } from "@/components/automation/NewAutomationDialog";
import { useWorkspaceStore } from "@/stores/workspaceStore";

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
      <NewAutomationDialog />
    </QueryClientProvider>,
  );
};

describe("NewAutomationDialog", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
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

  const pickLevel = async (user: UserEvent, domain: string, name: string) => {
    await user.click(await screen.findByRole("button", { name: new RegExp(`^${domain} access:`) }));
    await user.click(await screen.findByRole("menuitemradio", { name: new RegExp(`^${name}`) }));
  };

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
    await pickLevel(user, "Tickets", "Write");
    await pickLevel(user, "Events", "Read");
    await user.click(screen.getByRole("button", { name: "Create automation" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations", {
      name: "Slack notifier",
      scopes: ["tickets:read", "tickets:write", "events:read"],
      workspace_id: "ws-1",
    });
    expect(await screen.findByText("Slack notifier created")).toBeInTheDocument();
    expect(screen.getByText("dep_secret_abc123")).toBeInTheDocument();
    expect(screen.getByText(/npx @nexul\/sdk init/)).toBeInTheDocument();
  });

  it("drops the minted token once the dialog closes, so reopening starts from the form", async () => {
    mocks.post.mockResolvedValue({ data: { automation: { id: "a1", name: "x" }, token: "dep_once" } });
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole("button", { name: "New automation" }));
    await user.type(screen.getByLabelText("Name"), "x");
    await pickLevel(user, "Tickets", "Read");
    await user.click(screen.getByRole("button", { name: "Create automation" }));
    expect(await screen.findByText("dep_once")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Done" }));
    expect(screen.queryByText("dep_once")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "New automation" }));
    expect(screen.getByRole("button", { name: "Create automation" })).toBeInTheDocument();
    expect(screen.queryByText("dep_once")).not.toBeInTheDocument();
  });

  it("setting a domain back to None removes its scopes from the submitted set", async () => {
    mocks.post.mockResolvedValue({ data: { automation: { id: "a1", name: "x" }, token: "t" } });
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole("button", { name: "New automation" }));
    await user.type(screen.getByLabelText("Name"), "x");
    await pickLevel(user, "Docs", "Read");
    await pickLevel(user, "Tickets", "Read");
    await pickLevel(user, "Docs", "None");
    await user.click(screen.getByRole("button", { name: "Create automation" }));

    expect(mocks.post).toHaveBeenCalledWith("/api/automations", { name: "x", scopes: ["tickets:read"], workspace_id: "ws-1" });
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
