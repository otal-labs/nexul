import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SignInAccountsSection } from "@/components/you/SignInAccountsSection";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, delete: mocks.del },
  errorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}));

vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));

const github = { provider: "github", login: "onik97", name: "Onik", avatar_url: "", created_at: "2026-01-01T00:00:00Z" };
const google = { provider: "google", login: "onik@example.com", name: "Onik", avatar_url: "", created_at: "2026-02-01T00:00:00Z" };

const mockGet = (identities: unknown[], status: Record<string, boolean>, appSlug = "") =>
  mocks.get.mockImplementation((url: string) => {
    if (url === "/api/auth/identities") return Promise.resolve({ data: { identities } });
    if (url === "/api/auth/bootstrap-status") return Promise.resolve({ data: { configured: true, ...status } });
    if (url === "/api/connectors/github/app-config") return Promise.resolve({ data: { configured: true, app_slug: appSlug } });
    return Promise.reject(new Error(`unexpected GET ${url}`));
  });

const renderSection = (route = "/settings/profile") => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[route]}>
        <SignInAccountsSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("SignInAccountsSection", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.del.mockReset();
    mocks.toastSuccess.mockClear();
    mocks.toastError.mockClear();
  });

  it("lists only the providers the instance turned on and marks the last sign-in", async () => {
    mockGet([github], { google_configured: true });
    renderSection();
    expect(await screen.findByText("@onik97")).toBeInTheDocument();
    expect(screen.getByText("Your only sign-in")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Unlink GitHub" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Link Google" })).toBeInTheDocument();
    expect(screen.queryByText("Discord")).not.toBeInTheDocument();
  });

  it("starts a link by sending the browser to the provider", async () => {
    mockGet([github], { google_configured: true });
    mocks.post.mockResolvedValue({ data: { url: "https://accounts.google.com/o/oauth2/v2/auth?state=link.x" } });
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Link Google" }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://accounts.google.com/o/oauth2/v2/auth?state=link.x"));
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/identities/link", { provider: "google" });
    vi.unstubAllGlobals();
  });

  it("unlinks after confirming, when another sign-in remains", async () => {
    mockGet([github, google], { google_configured: true });
    mocks.del.mockResolvedValue({});
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Unlink Google" }));
    await user.click(screen.getByRole("button", { name: "Unlink" }));
    await waitFor(() => expect(mocks.del).toHaveBeenCalledWith("/api/auth/identities/google"));
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Google unlinked");
  });

  it("toasts the link callback's outcome once", async () => {
    mockGet([github, google], { google_configured: true });
    renderSection("/settings/profile?provider=google&error=conflict%3A+already+linked+to+another+user");
    await screen.findByText("onik@example.com");
    await waitFor(() => expect(mocks.toastError).toHaveBeenCalledWith("Google: conflict: already linked to another user"));
    expect(mocks.toastError).toHaveBeenCalledTimes(1);
  });

  it("offers someone with GitHub linked the App's install page on their own account", async () => {
    mockGet([github], {}, "nexul-app");
    renderSection();

    expect(await screen.findByRole("link", { name: /let nexul deploy your repositories/i })).toHaveAttribute(
      "href",
      "https://github.com/apps/nexul-app/installations/new",
    );
    expect(screen.getByText("Installs Nexul's GitHub App on your account; pick which repositories.")).toBeInTheDocument();
  });

  it("offers no install link while the instance's GitHub App has no slug", async () => {
    mockGet([github], {});
    renderSection();

    expect(await screen.findByText("@onik97")).toBeInTheDocument();
    await waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/connectors/github/app-config"));
    expect(screen.queryByRole("link", { name: /let nexul deploy your repositories/i })).not.toBeInTheDocument();
  });
});
