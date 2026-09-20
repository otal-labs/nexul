import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { InvitePreviewPage } from "@/pages/InvitePreviewPage";
import { useSessionStore } from "@/stores/sessionStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));
const originalLocation = window.location;
vi.mock("@/api/client", () => ({ api: { post: mocks.post, get: mocks.get, patch: vi.fn(), delete: vi.fn() }, errorMessage: vi.fn() }));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const renderPage = () => render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><MemoryRouter initialEntries={[`/invite${window.location.hash}`]}><Routes><Route path="/invite" element={<InvitePreviewPage />} /><Route path="/" element={<div>home</div>} /></Routes></MemoryRouter></QueryClientProvider>);

describe("InvitePreviewPage", () => {
  afterEach(() => {
    Object.defineProperty(window, "location", { configurable: true, value: originalLocation });
  });
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.get.mockReset();
    mocks.get.mockResolvedValue({ data: { configured: true, google_configured: false, discord_configured: false } });
    useSessionStore.setState({ token: null, isLoggedIn: false });
    window.history.replaceState(null, "", "/invite#raw-token");
  });

  it("clears the raw fragment immediately and renders a public preview", async () => {
    mocks.post.mockResolvedValue({ data: { instance_name: "Private Nexul", grants: [{ workspace_id: "ws-1", workspace_name: "Engineering", role_id: "r-1", role_name: "Editor", allow: [], deny: [] }], expires_at: "2026-09-21T00:00:00Z", providers: ["github"] } });
    renderPage();
    expect(window.location.hash).toBe("");
    expect(await screen.findByRole("heading", { name: /join private nexul/i })).toBeInTheDocument();
    expect(screen.getByText("Engineering")).toBeInTheDocument();
  });

  it("shows the same generic invalid state for a rejected preview", async () => {
    mocks.post.mockRejectedValue(new Error("secret backend reason"));
    renderPage();
    expect(await screen.findByText("This invitation is invalid or has expired.")).toBeInTheDocument();
    expect(screen.queryByText("secret backend reason")).not.toBeInTheDocument();
  });

  it("posts the provider and token when a configured provider is chosen", async () => {
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [], expires_at: "later", providers: ["github"] } }).mockResolvedValueOnce({ data: { url: "https://oauth.example/authorize" } });
    const user = userEvent.setup();
    renderPage();
    Object.defineProperty(window, "location", { configurable: true, value: { pathname: "/invite", search: "", hash: "", assign: vi.fn() } });
    await user.click(await screen.findByRole("button", { name: /continue with github/i }));
    expect(api.post).toHaveBeenCalledWith("/api/invitations/oauth", { provider: "github", token: "raw-token" });
  });

  it("shows authenticated grants and redeems only after explicit acceptance", async () => {
    useSessionStore.setState({ token: "session", isLoggedIn: true });
    window.history.replaceState(null, "", "/invite#acceptance-token");
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [{ workspace_id: "ws-1", workspace_name: "Engineering", role_id: "r-1", role_name: "Editor", allow: ["docs:read"], deny: [] }], expires_at: "later", providers: [], acceptance_token: "acceptance-token" } }).mockResolvedValueOnce({ data: { token: "new-session", workspace_ids: ["ws-1"] } });
    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("Allow: docs:read")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole("button", { name: "Accept invitation" }));
    expect(mocks.post).toHaveBeenNthCalledWith(2, "/api/invitations/redeem", { acceptance_token: "acceptance-token" });
    expect(useSessionStore.getState().token).toBe("new-session");
  });

  it("renders the generic invalid state for malformed percent encoding", async () => {
    window.history.replaceState(null, "", "/invite#%E0%A4%A");
    renderPage();
    expect(await screen.findByText("This invitation is invalid or has expired.")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("declines without redeeming and returns to the home route", async () => {
    useSessionStore.setState({ token: "session", isLoggedIn: true });
    window.history.replaceState(null, "", "/invite#acceptance-token");
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [], expires_at: "later", providers: [], acceptance_token: "acceptance-token" } });
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: "Decline" }));
    expect(await screen.findByText("home")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });
});
