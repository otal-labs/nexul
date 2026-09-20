import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AccountsSection } from "@/components/settings/AccountsSection";

const mocks = vi.hoisted(() => ({ get: vi.fn(), patch: vi.fn(), delete: vi.fn() }));
vi.mock("@/api/client", () => ({ api: mocks, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/useConfirmationDialog", () => ({ useConfirmationDialog: () => ({ open: vi.fn().mockResolvedValue(true) }) }));

const renderSection = () => render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><AccountsSection /></QueryClientProvider>);

describe("AccountsSection", () => {
  it("lists account status and disables an active account after confirmation", async () => {
    mocks.get.mockResolvedValue({ data: [{ id: "u-1", provider: "github", login: "alice", name: "Alice", avatar_url: "", status: "active", can_create_workspace: false, created_at: "" }] });
    mocks.patch.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();
    expect(await screen.findByText("active")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Disable" }));
    expect(mocks.patch).toHaveBeenCalledWith("/api/auth/accounts/u-1", { status: "disabled" });
  });

  it("restores a removed account", async () => {
    mocks.get.mockResolvedValue({ data: [{ id: "u-2", provider: "github", login: "bob", name: "Bob", avatar_url: "", status: "removed", can_create_workspace: false, created_at: "" }] });
    mocks.patch.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Restore" }));
    expect(mocks.patch).toHaveBeenCalledWith("/api/auth/accounts/u-2", { status: "active" });
  });

  it("removes an active account after the destructive confirmation", async () => {
    mocks.get.mockResolvedValue({ data: [{ id: "u-3", provider: "github", login: "carol", name: "Carol", avatar_url: "", status: "active", can_create_workspace: false, created_at: "" }] });
    mocks.delete.mockResolvedValue({ data: {} });
    const user = userEvent.setup();
    renderSection();
    await user.click(await screen.findByRole("button", { name: "Remove account" }));
    await user.click(screen.getByRole("button", { name: "Confirm" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/auth/accounts/u-3");
  });
});
