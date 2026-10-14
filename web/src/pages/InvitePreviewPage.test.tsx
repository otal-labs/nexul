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

const GONE = "This invitation has been deleted or has expired";

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

  it("clears the raw fragment immediately and renders a public preview of each role and what it means, with no projects", async () => {
    mocks.post.mockResolvedValue({ data: { instance_name: "Private Nexul", grants: [{ workspace_id: "ws-1", workspace_name: "Engineering", role_id: "r-1", role_name: "Editor", roles: [{ id: "r-1", name: "Editor", description: "Edits tickets and docs." }, { id: "r-2", name: "Frontend" }], projects: [] }], expires_at: "2026-09-21T00:00:00Z", providers: ["github"] } });
    renderPage();
    expect(window.location.hash).toBe("");
    expect(await screen.findByRole("heading", { name: /join private nexul/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Engineering" })).toBeInTheDocument();
    expect(screen.getAllByRole("listitem").map((role) => role.textContent)).toEqual(["EditorEdits tickets and docs.", "Frontend"]);
    expect(screen.getByText("Sign in to accept")).toBeInTheDocument();
    expect(screen.queryByText(/projects?:|No projects yet/)).not.toBeInTheDocument();
    expect(screen.queryByText("You may be added to more projects later.")).not.toBeInTheDocument();
  });

  it("says only that the link is gone when the server can't find it", async () => {
    mocks.post.mockRejectedValue(Object.assign(new Error("secret backend reason"), { response: { status: 404, data: { code: "NOT_FOUND", message: "secret backend reason" } } }));
    renderPage();
    expect(await screen.findByRole("heading", { name: GONE })).toBeInTheDocument();
    expect(screen.queryByText(/secret backend reason/)).not.toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("says the link is gone when the provider round trip found it gone, without reading the fragment as a link", async () => {
    window.history.replaceState(null, "", "/invite#refused=gone");
    renderPage();
    expect(await screen.findByRole("heading", { name: GONE })).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("turns a signed-in person away, naming them from the server's refusal", async () => {
    useSessionStore.setState({ token: "session", isLoggedIn: true });
    mocks.post.mockRejectedValue(Object.assign(new Error("conflict"), { response: { status: 409, data: { code: "existing_account", message: "conflict", details: { provider: "github", login: "lena", name: "Lena Fischer", avatar_url: "" } } } }));
    renderPage();
    expect(await screen.findByRole("heading", { name: "You already have an account here" })).toBeInTheDocument();
    expect(screen.getByText("Lena Fischer")).toBeInTheDocument();
    expect(screen.getByText(/@lena · GitHub/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /go to/i })).toHaveAttribute("href", "/");
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });

  it("turns away an account the provider round trip found, without reading the fragment as a link", async () => {
    window.history.replaceState(null, "", "/invite#login=lena&name=Lena+Fischer&provider=github&refused=existing_account");
    renderPage();
    expect(await screen.findByRole("heading", { name: "You already have an account here" })).toBeInTheDocument();
    expect(screen.getByText("Lena Fischer")).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("posts the provider and token when a configured provider is chosen", async () => {
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [], expires_at: "later", providers: ["github"] } }).mockResolvedValueOnce({ data: { url: "https://oauth.example/authorize" } });
    const user = userEvent.setup();
    renderPage();
    Object.defineProperty(window, "location", { configurable: true, value: { pathname: "/invite", search: "", hash: "", assign: vi.fn() } });
    await user.click(await screen.findByRole("button", { name: /continue with github/i }));
    expect(api.post).toHaveBeenCalledWith("/api/invitations/oauth", { provider: "github", token: "raw-token" });
  });

  it("names after sign-in the projects each workspace's roles reach, and who is accepting", async () => {
    window.history.replaceState(null, "", "/invite#acceptance-token=acceptance-token");
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [
      { workspace_id: "ws-1", workspace_name: "Northwind", role_id: "r-1", role_name: "Client", roles: [{ id: "r-1", name: "Client", description: "Reads the projects you're added to." }], projects: [{ id: "p-1", name: "Website relaunch" }, { id: "p-2", name: "Mobile app" }] },
      { workspace_id: "ws-2", workspace_name: "Acme Internal", role_id: "r-2", role_name: "Reviewer", roles: [{ id: "r-2", name: "Reviewer" }], projects: [] },
    ], expires_at: "later", authenticated_user: { id: "", provider: "github", login: "lena", name: "Lena Fischer" }, acceptance_token: "acceptance-token" } });
    renderPage();
    expect(await screen.findByText("Website relaunch, Mobile app")).toBeInTheDocument();
    expect(screen.getByText("2 projects:")).toBeInTheDocument();
    expect(screen.getByText("No projects yet")).toBeInTheDocument();
    expect(screen.getByText("You may be added to more projects later.")).toBeInTheDocument();
    expect(screen.getByText("Lena Fischer")).toBeInTheDocument();
    expect(screen.getByText(/@lena · GitHub/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Decline" })).toBeInTheDocument();
    expect(screen.queryByText("Sign in to accept")).not.toBeInTheDocument();
  });

  it("shows authenticated grants and redeems only after explicit acceptance", async () => {
    window.history.replaceState(null, "", "/invite#acceptance-token=acceptance-token");
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [{ workspace_id: "ws-1", workspace_name: "Engineering", role_id: "r-1", role_name: "Editor", roles: [{ id: "r-1", name: "Editor" }] }], expires_at: "later", providers: [], acceptance_token: "acceptance-token" } }).mockResolvedValueOnce({ data: { token: "new-session", workspace_ids: ["ws-1"] } });
    const user = userEvent.setup();
    renderPage();
    expect(await screen.findByText("Editor")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole("button", { name: "Accept invitation" }));
    expect(mocks.post).toHaveBeenNthCalledWith(2, "/api/invitations/redeem", { acceptance_token: "acceptance-token" });
    expect(useSessionStore.getState().token).toBe("new-session");
  });

  it("lands a signed-out invitee back from the provider on the acceptance, never on the invitation preview", async () => {
    window.history.replaceState(null, "", "/invite#acceptance-token=hand-off");
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [], expires_at: "later", providers: [], acceptance_token: "hand-off" } });
    renderPage();
    expect(await screen.findByRole("button", { name: "Accept invitation" })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/invitations/acceptance", { acceptance_token: "hand-off" });
    expect(mocks.post).not.toHaveBeenCalledWith("/api/invitations/preview", expect.anything());
  });

  it("tries again with the link it already read, since the address bar no longer has it", async () => {
    mocks.post.mockRejectedValueOnce(new Error("network")).mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [], expires_at: "later", providers: ["github"] } });
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: "Try again" }));
    expect(await screen.findByRole("heading", { name: /join private nexul/i })).toBeInTheDocument();
    expect(mocks.post).toHaveBeenLastCalledWith("/api/invitations/preview", { token: "raw-token" });
  });

  it("says a link with no token is gone, with no check still spinning", async () => {
    window.history.replaceState(null, "", "/invite");
    renderPage();
    expect(await screen.findByRole("heading", { name: GONE })).toBeInTheDocument();
    expect(screen.queryByText("Checking invitation…")).not.toBeInTheDocument();
  });

  it("says the link is gone for malformed percent encoding", async () => {
    window.history.replaceState(null, "", "/invite#%E0%A4%A");
    renderPage();
    expect(await screen.findByRole("heading", { name: GONE })).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("declines without redeeming and returns to the home route", async () => {
    window.history.replaceState(null, "", "/invite#acceptance-token=acceptance-token");
    mocks.post.mockResolvedValueOnce({ data: { instance_name: "Private Nexul", grants: [], expires_at: "later", providers: [], acceptance_token: "acceptance-token" } });
    const user = userEvent.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: "Decline" }));
    expect(await screen.findByText("home")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledTimes(1);
  });
});
