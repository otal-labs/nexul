import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OAuthAuthorizePage } from "@/pages/OAuthAuthorizePage";
import { useSessionStore } from "@/stores/sessionStore";
import { useSignInReturnStore } from "@/stores/signInReturnStore";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), assign: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: (error: unknown) => (error as Error).message,
}));

const search = "?client_id=c1&redirect_uri=https%3A%2F%2Fagent.example.com%2Fcb&response_type=code&code_challenge=abc&code_challenge_method=S256&state=xyz";

const renderPage = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/oauth/authorize${search}`]}>
        <Routes>
          <Route path="/oauth/authorize" element={<OAuthAuthorizePage />} />
          <Route path="/login" element={<p>sign-in page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("OAuthAuthorizePage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.assign.mockReset();
    Object.defineProperty(window, "location", { configurable: true, value: { assign: mocks.assign, host: "nexul.example.com" } });
    useSignInReturnStore.setState({ path: null });
    useSessionStore.setState({ token: "ses_1", isLoggedIn: true });
    mocks.get.mockResolvedValue({ data: { client_name: "Globex Agent", redirect_uri: "https://agent.example.com/cb" } });
  });

  it("signed out, keeps the request and sends the person to sign in first", async () => {
    useSessionStore.setState({ token: null, isLoggedIn: false });
    renderPage();
    expect(await screen.findByText("sign-in page")).toBeInTheDocument();
    expect(useSignInReturnStore.getState().path).toBe(`/oauth/authorize${search}`);
    expect(mocks.get).not.toHaveBeenCalledWith(expect.stringContaining("/api/auth/oauth"));
  });

  it("names the app and where it goes back to, and allowing follows the server's redirect", async () => {
    mocks.post.mockResolvedValue({ data: { redirect_to: "https://agent.example.com/cb?code=k&state=xyz" } });
    renderPage();
    expect(await screen.findByRole("heading", { name: "Allow Globex Agent?" })).toBeInTheDocument();
    expect(screen.getByText("agent.example.com")).toBeInTheDocument();
    expect(mocks.get).toHaveBeenCalledWith(`/api/auth/oauth/authorize${search}`);

    await userEvent.setup().click(screen.getByRole("button", { name: "Allow" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/oauth/authorize", {
      client_id: "c1", redirect_uri: "https://agent.example.com/cb", response_type: "code",
      code_challenge: "abc", code_challenge_method: "S256", state: "xyz", approve: true,
    });
    expect(mocks.assign).toHaveBeenCalledWith("https://agent.example.com/cb?code=k&state=xyz");
    expect(await screen.findByText("You can close this tab.")).toBeInTheDocument();
  });

  it("denying answers approve false, and a script URL is never followed", async () => {
    mocks.post.mockResolvedValue({ data: { redirect_to: "javascript:alert(1)" } });
    renderPage();
    await userEvent.setup().click(await screen.findByRole("button", { name: "Deny" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/oauth/authorize", expect.objectContaining({ approve: false }));
    expect(mocks.assign).not.toHaveBeenCalled();
  });

  it("shows why the server refused the request instead of a consent", async () => {
    mocks.get.mockRejectedValue(new Error("redirect_uri is not one this app registered"));
    renderPage();
    expect(await screen.findByText("redirect_uri is not one this app registered")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Allow" })).not.toBeInTheDocument();
  });
});
