import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, afterEach, describe, expect, it, vi } from "vitest";

import { LoginPage } from "@/pages/LoginPage";
import { useSessionStore } from "@/stores/sessionStore";

const mocks = vi.hoisted(() => ({
  post: vi.fn(),
  get: vi.fn(),
  errorMessage: vi.fn((error: unknown) => {
    const body = (error as { response?: { data?: { message?: string } } })?.response?.data;
    return body?.message ?? "Something went wrong";
  }),
}));

vi.mock("@/api/client", () => ({
  api: { post: mocks.post, get: mocks.get },
  errorMessage: mocks.errorMessage,
  API_BASE_URL: "http://localhost:8080",
  joinAPIURL: (path: string) => `http://localhost:8080${path}`,
}));

const renderPage = (initialEntries: string[] = ["/login"]) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={initialEntries}>
        <LoginPage />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

describe("LoginPage", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.get.mockReset();
    mocks.get.mockResolvedValue({ data: { configured: true, google_configured: false } });
    mocks.errorMessage.mockClear();
    useSessionStore.setState({ token: null, isLoggedIn: false });
    vi.stubEnv("VITE_API_URL", "http://localhost:8080");
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("renders the sign-in button", () => {
    renderPage();
    expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
  });

  it("redirects to the GitHub OAuth start URL", async () => {
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign: assignMock },
    });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderPage();
    await user.click(screen.getByRole("button", { name: /continue with github/i }));
    expect(assignMock).toHaveBeenCalledWith("http://localhost:8080/auth/github");
  });

  it("offers Google only when the instance has it configured", async () => {
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign: assignMock },
    });
    mocks.get.mockResolvedValue({ data: { configured: true, google_configured: true } });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: /continue with google/i }));
    expect(assignMock).toHaveBeenCalledWith("http://localhost:8080/auth/google");
  });

  it("offers a way back into bootstrap while no user has logged in yet", async () => {
    mocks.get.mockResolvedValue({ data: { configured: true, reconfigurable: true } });
    renderPage();
    expect(await screen.findByRole("link", { name: /set up the github app again/i })).toHaveAttribute("href", "/setup");
  });

  it("hides the bootstrap link once someone has logged in", async () => {
    mocks.get.mockResolvedValue({ data: { configured: true, reconfigurable: false } });
    renderPage();
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/auth/bootstrap-status"));
    expect(screen.queryByRole("link", { name: /set up the github app again/i })).not.toBeInTheDocument();
  });

  it("offers Discord when configured", async () => {
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign: assignMock },
    });
    mocks.get.mockResolvedValue({ data: { configured: true, discord_configured: true } });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: /continue with discord/i }));
    expect(assignMock).toHaveBeenCalledWith("http://localhost:8080/auth/discord");
    expect(screen.queryByRole("button", { name: /continue with google/i })).not.toBeInTheDocument();
  });

  it("hides the Google button when not configured", async () => {
    renderPage();
    await screen.findByRole("button", { name: /continue with github/i });
    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalledWith("/api/auth/bootstrap-status"));
    expect(screen.queryByRole("button", { name: /continue with google/i })).not.toBeInTheDocument();
  });

  it("exchanges the OAuth code and stores the session", async () => {
    let resolvePost: (value: { data: { token: string } }) => void;
    mocks.post.mockReturnValue(
      new Promise((resolve) => {
        resolvePost = resolve;
      }),
    );
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign: vi.fn() },
    });
    renderPage(["/login?code=abc123"]);
    await screen.findByText(/completing sign in/i);
    resolvePost!({ data: { token: "tok-oauth" } });
    await vi.waitFor(() => {
      expect(useSessionStore.getState().isLoggedIn).toBe(true);
    });
    expect(useSessionStore.getState().token).toBe("tok-oauth");
    expect(mocks.post).toHaveBeenCalledWith("/auth/callback", { code: "abc123" });
  });

  it("logs in directly when the server redirects with a token", async () => {
    renderPage(["/login?token=tok-server"]);
    await vi.waitFor(() => {
      expect(useSessionStore.getState().isLoggedIn).toBe(true);
    });
    expect(useSessionStore.getState().token).toBe("tok-server");
    expect(mocks.post).not.toHaveBeenCalled();
  });

  it("shows an error when the code exchange fails", async () => {
    mocks.post.mockRejectedValue({ response: { data: { message: "OAuth failed", code: "oauth" } } });
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign: vi.fn(), search: "?code=bad" },
    });
    renderPage(["/login?code=bad"]);
    expect(await screen.findByText("OAuth failed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
  });
});
