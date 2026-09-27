import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SetupFlow } from "@/components/setup/SetupFlow";
import { useSetupPassStore } from "@/stores/setupPassStore";
import type { BootstrapStatus } from "@/models/User";

const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), put: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post, put: mocks.put },
  errorMessage: (e: { response?: { data?: { message?: string } } }) => e?.response?.data?.message ?? "Something went wrong",
  joinAPIURL: (path: string) => path,
}));

const DOMAIN = "https://deploy.example.com";

const renderFlow = (status: BootstrapStatus) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SetupFlow status={status} />
    </QueryClientProvider>,
  );
};

const withPass = (code: string | null = "nxs_abc") =>
  useSetupPassStore.setState({ token: "pass-1", expiresAt: "2999-01-01T00:00:00Z", code });

describe("SetupFlow", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.post.mockReset();
    mocks.put.mockReset();
    useSetupPassStore.setState({ token: null, expiresAt: null, code: null });
  });

  it("shows sign-in instead of any setup screen once a user exists", () => {
    withPass();
    renderFlow({ configured: true, setup_open: false });

    expect(screen.getByRole("heading", { name: /already set up/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /sign in/i })).toHaveAttribute("href", "/login");
    expect(screen.queryByLabelText(/setup code/i)).not.toBeInTheDocument();
  });

  it("asks for the setup code before anything else", () => {
    renderFlow({ configured: false, setup_open: true, instance_url: DOMAIN });

    expect(screen.getByRole("heading", { name: /enter the setup code/i })).toBeInTheDocument();
    expect(screen.getByText("sudo nexul status")).toBeInTheDocument();
  });

  it("shows the domain step with its three paths and no skip on a server install", () => {
    withPass();
    renderFlow({ configured: false, setup_open: true });

    expect(screen.getByRole("heading", { name: /give nexul a domain/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /cloudflare tunnel/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /reverse proxy/i })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /i already have https/i })).toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /bare public address/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /skip/i })).not.toBeInTheDocument();
  });

  it("stores this localhost origin on a desktop install instead of asking for a domain", async () => {
    let release: (value: unknown) => void = () => {};
    mocks.put.mockImplementation(() => new Promise((resolve) => (release = resolve)));
    withPass();
    renderFlow({ configured: false, setup_open: true, local: true });

    expect(screen.queryByRole("radio", { name: /cloudflare tunnel/i })).not.toBeInTheDocument();
    expect(await screen.findByText(/answers as nexul over https/i)).toHaveTextContent(window.location.origin);
    expect(mocks.put).toHaveBeenCalledTimes(1);
    expect(mocks.put).toHaveBeenCalledWith("/api/setup/instance-url", { url: window.location.origin });
    release({ data: { instance_url: window.location.origin } });
  });

  it("shows the reason and a retry when the desktop URL is refused", async () => {
    mocks.put.mockRejectedValue({ response: { status: 400, data: { message: "http is only allowed on local installs" } } });
    withPass();
    renderFlow({ configured: false, setup_open: true, local: true });

    expect(await screen.findByRole("alert")).toHaveTextContent(/only allowed on local installs/i);
    expect(screen.getByRole("button", { name: /try again/i })).toBeInTheDocument();
  });

  it("hands off to the domain with the code in the fragment when still on IP:port", () => {
    withPass("nxs_abc");
    renderFlow({ configured: false, setup_open: true, instance_url: DOMAIN });

    expect(screen.getByText(DOMAIN)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /continue there/i })).toHaveAttribute(
      "href",
      `${DOMAIN}/setup#code=nxs_abc`,
    );
  });

  it("hands off without the code after a reload, and says it will be asked again", () => {
    withPass(null);
    renderFlow({ configured: false, setup_open: true, instance_url: DOMAIN });

    expect(screen.getByRole("link", { name: /continue there/i })).toHaveAttribute("href", `${DOMAIN}/setup`);
    expect(screen.getByText(/enter the setup code once more/i)).toBeInTheDocument();
  });

  it("shows the GitHub step with the instance URL fixed once on the domain", () => {
    withPass();
    renderFlow({ configured: false, setup_open: true, instance_url: window.location.origin });

    expect(screen.getByRole("heading", { name: /connect a github app/i })).toBeInTheDocument();
    expect(screen.queryByLabelText(/instance url/i)).not.toBeInTheDocument();
    expect(screen.getByText(`${window.location.origin}/auth/callback`)).toBeInTheDocument();
  });
});
