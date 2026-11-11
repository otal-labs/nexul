import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
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

const passkeys = vi.hoisted(() => ({ supported: vi.fn(() => true), get: vi.fn() }));

vi.mock("@/lib/passkeys", async (importActual) => ({
  ...(await importActual<typeof import("@/lib/passkeys")>()),
  passkeysSupported: passkeys.supported,
  getPasskey: passkeys.get,
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

  it.each([
    ["google", "discord"],
    ["discord", "google"],
  ])("offers %s only when the instance has it configured, and not %s", async (offered, other) => {
    const assignMock = vi.fn();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: { assign: assignMock },
    });
    mocks.get.mockResolvedValue({ data: { configured: true, [`${offered}_configured`]: true } });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderPage();
    await user.click(await screen.findByRole("button", { name: new RegExp(`continue with ${offered}`, "i") }));
    expect(assignMock).toHaveBeenCalledWith(`http://localhost:8080/auth/${offered}`);
    expect(screen.queryByRole("button", { name: new RegExp(`continue with ${other}`, "i") })).not.toBeInTheDocument();
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
    await screen.findByText(/signing you in/i);
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

  it("says an invitation was declined, as a note rather than an error", async () => {
    renderPage(["/login?declined=1"]);
    expect(await screen.findByRole("status")).toHaveTextContent("Invitation declined");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
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

  it("explains an invitation-required sign-in above the buttons", async () => {
    renderPage(["/login?error=invitation_required"]);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This instance is invite-only. Ask someone on it for an invitation link.",
    );
    expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
  });

  it("falls back to a generic line for an error code it does not know", async () => {
    renderPage(["/login?error=something_new"]);
    expect(await screen.findByRole("alert")).toHaveTextContent("Sign-in didn't work. Try again.");
  });

  it("shows no error when the page is opened plainly", () => {
    renderPage();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});

describe("LoginPage passkey sign-in", () => {
  beforeEach(() => {
    mocks.post.mockReset();
    mocks.get.mockReset();
    passkeys.supported.mockReturnValue(true);
    useSessionStore.setState({ token: null, isLoggedIn: false });
  });

  it("is offered only where the server says passkeys work", async () => {
    mocks.get.mockResolvedValue({ data: { configured: true, passkeys: false } });
    renderPage();
    await screen.findByRole("button", { name: /continue with github/i });
    expect(screen.queryByRole("button", { name: "Sign in with a passkey" })).not.toBeInTheDocument();
  });

  it("signs in with the session the passkey's assertion earns", async () => {
    mocks.get.mockResolvedValue({ data: { configured: true, passkeys: true } });
    mocks.post.mockImplementation((url: string) => {
      if (url === "/api/auth/passkeys/login/start") return Promise.resolve({ data: { ceremony: "pkc_1", options: { challenge: "abc" } } });
      return Promise.resolve({ data: { token: "ses_passkey" } });
    });
    passkeys.get.mockResolvedValue({ id: "cred-1" });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Sign in with a passkey" }));

    await waitFor(() => expect(useSessionStore.getState().token).toBe("ses_passkey"));
    expect(mocks.post).toHaveBeenCalledWith("/api/auth/passkeys/login/finish", { ceremony: "pkc_1", credential: { id: "cred-1" } });
  });

  it("shows why a passkey was refused", async () => {
    mocks.get.mockResolvedValue({ data: { configured: true, passkeys: true } });
    mocks.post.mockImplementation((url: string) => {
      if (url === "/api/auth/passkeys/login/start") return Promise.resolve({ data: { ceremony: "pkc_1", options: {} } });
      return Promise.reject({ response: { data: { message: "that passkey isn't recognised here" } } });
    });
    passkeys.get.mockResolvedValue({ id: "cred-1" });
    const user = (await import("@testing-library/user-event")).default.setup();
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Sign in with a passkey" }));

    expect(await screen.findByText("that passkey isn't recognised here")).toBeInTheDocument();
  });

  describe("single sign-on", () => {
    const withSso = (sso: { active: boolean; name: string; only: boolean }) =>
      mocks.get.mockImplementation((url: string) =>
        Promise.resolve({ data: url === "/api/auth/sso" ? sso : { configured: true, google_configured: true, passkeys: true } }),
      );

    it("shows the company's button first and leaves the other methods", async () => {
      withSso({ active: true, name: "Acme SSO", only: false });
      const assignMock = vi.fn();
      Object.defineProperty(window, "location", { configurable: true, value: { assign: assignMock } });
      const user = (await import("@testing-library/user-event")).default.setup();
      renderPage();

      await user.click(await screen.findByRole("button", { name: "Sign in with Acme SSO" }));

      expect(assignMock).toHaveBeenCalledWith("http://localhost:8080/auth/oidc");
      expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /continue with google/i })).toBeInTheDocument();
    });

    it("hides every other method while it is the only sign-in, behind a link to the Owner's passkey", async () => {
      withSso({ active: true, name: "Acme SSO", only: true });
      renderPage(["/login?error=sso_only"]);

      await screen.findByRole("button", { name: "Sign in with Acme SSO" });
      expect(await screen.findByRole("link", { name: "Instance Owner? Use your passkey" })).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /continue with/i })).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Sign in with a passkey" })).not.toBeInTheDocument();
      expect(screen.getByText(/signs in through single sign-on only/i)).toBeInTheDocument();
    });

    it("offers the Owner's passkey once they follow the link", async () => {
      withSso({ active: true, name: "Acme SSO", only: true });
      renderPage(["/login?owner"]);

      expect(await screen.findByRole("button", { name: "Sign in with a passkey" })).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /continue with/i })).not.toBeInTheDocument();
    });

    it("shows nothing of it while the license is off, whatever was saved", async () => {
      withSso({ active: false, name: "", only: false });
      renderPage();

      expect(await screen.findByRole("button", { name: "Sign in with a passkey" })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /^Sign in with (?!a passkey)/ })).not.toBeInTheDocument();
    });
  });
});
