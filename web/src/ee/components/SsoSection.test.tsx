import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SsoSection } from "@/ee/components/SsoSection";
import { getSsoKey, getSsoPublicKey, ssoFollower } from "@/ee/hooks/SsoHooks";
import type { SsoSettings } from "@/ee/models/Sso";
import { followFrame, isStale, seeded } from "@/test/followFrame";
import { pickOption } from "@/test/pickOption";

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  delete: vi.fn(),
  errorMessage: vi.fn((err: unknown) => (err as Error).message),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, post: mocks.post, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const roles = [
  { id: "role-owner", name: "Owner", is_owner_role: true },
  { id: "role-member", name: "Member", is_owner_role: false },
];

const sso = (over: Partial<SsoSettings> = {}): SsoSettings => ({
  unlocked: true,
  configured: true,
  issuer_url: "https://login.example.com",
  client_id: "client-1",
  client_secret_set: true,
  display_name: "Acme SSO",
  join_role_id: "role-member",
  only: false,
  redirect_uri: "https://nexul.example.com/auth/oidc/callback",
  active: true,
  ...over,
});

const unset = (over: Partial<SsoSettings> = {}) =>
  sso({ configured: false, issuer_url: "", client_id: "", client_secret_set: false, display_name: "", join_role_id: "", active: false, ...over });

const serve = (settings: SsoSettings) =>
  mocks.get.mockImplementation((url: string) => Promise.resolve({ data: url === "/api/instance/sso" ? settings : roles }));

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SsoSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("SsoSection", () => {
  it("says it needs a license, keeps the saved settings in view and offers no way to change them, only to remove them", async () => {
    serve(sso({ unlocked: false, active: false, only: true }));
    renderSection();

    expect(await screen.findByText("Single sign-on needs a license key")).toBeInTheDocument();
    expect(screen.getByText(/saved settings stay/)).toBeInTheDocument();
    expect(screen.getByText("Needs a license")).toBeInTheDocument();
    expect(screen.getByText("Acme SSO")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^(Set up|Edit) provider$/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove provider" }), "a lapsed key never traps a provider in place").toBeInTheDocument();
  });

  it("offers setup once the license unlocks it", async () => {
    serve(unset());
    renderSection();

    expect(await screen.findByRole("button", { name: "Set up provider" })).toBeInTheDocument();
    expect(screen.getByText("Off")).toBeInTheDocument();
    expect(screen.getByText("https://nexul.example.com/auth/oidc/callback")).toBeInTheDocument();
  });

  it("saves only after both checks pass, never sends the saved secret, and shows what the server refuses", async () => {
    serve(unset());
    mocks.post.mockImplementation((_url: string, _data: unknown, config: { params: { check: string } }) =>
      config.params.check === "issuer" ? Promise.resolve({ data: { detail: "Found the sign-in endpoints at login.example.com" } }) : Promise.resolve({ data: undefined }),
    );
    mocks.put.mockRejectedValueOnce(new Error("join_role_id must be a role of the default workspace other than Owner")).mockResolvedValue({ data: sso() });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Set up provider" }));
    await user.type(screen.getByLabelText("Issuer URL"), "https://login.example.com");
    await user.type(screen.getByLabelText("Client ID"), "client-1");
    await user.type(screen.getByLabelText("Client secret"), "s3cret");
    await user.type(screen.getByLabelText("Name on the button"), "Acme SSO");
    await pickOption(user, "New people join as", "Member");

    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Verify" }));
    expect(await screen.findByText("Found the sign-in endpoints at login.example.com")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/sso/verify", expect.objectContaining({ issuer_url: "https://login.example.com" }), { params: { check: "issuer" } });
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/sso/verify", expect.anything(), { params: { check: "secret" } });

    await user.click(await screen.findByRole("button", { name: "Confirm" }));
    expect(await screen.findByText(/other than Owner/)).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(mocks.put).toHaveBeenLastCalledWith("/api/instance/sso", {
      issuer_url: "https://login.example.com",
      client_id: "client-1",
      client_secret: "s3cret",
      display_name: "Acme SSO",
      join_role_id: "role-member",
    });
  });

  it("holds Confirm back while a check fails, and says which", async () => {
    serve(unset());
    mocks.post.mockImplementation((_url: string, _data: unknown, config: { params: { check: string } }) =>
      config.params.check === "issuer" ? Promise.reject(new Error("cannot reach the provider at https://nope.example.com")) : Promise.resolve({ data: undefined }),
    );
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Set up provider" }));
    await user.type(screen.getByLabelText("Issuer URL"), "https://nope.example.com");
    await user.type(screen.getByLabelText("Client ID"), "client-1");
    await user.type(screen.getByLabelText("Client secret"), "s3cret");
    await user.type(screen.getByLabelText("Name on the button"), "Acme");
    await pickOption(user, "New people join as", "Member");
    await user.click(screen.getByRole("button", { name: "Verify" }));

    expect(await screen.findByText("cannot reach the provider at https://nope.example.com")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("keeps the saved secret when editing, and leaves the Owner role out of the choices", async () => {
    serve(sso());
    mocks.post.mockResolvedValue({ data: undefined });
    mocks.put.mockResolvedValue({ data: sso({ display_name: "Acme" }) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Edit provider" }));
    await user.click(screen.getByRole("combobox", { name: "New people join as" }));
    expect(screen.queryByRole("option", { name: "Owner" })).not.toBeInTheDocument();
    await user.keyboard("{Escape}");
    await user.clear(screen.getByLabelText("Name on the button"));
    await user.type(screen.getByLabelText("Name on the button"), "Acme");
    await user.click(screen.getByRole("button", { name: "Verify" }));
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalled());
    expect(mocks.put).toHaveBeenCalledWith("/api/instance/sso", expect.objectContaining({ client_secret: "", display_name: "Acme" }));
  });

  it("flips SSO-only on the switch and shows why the server turned it down", async () => {
    serve(sso());
    mocks.put.mockRejectedValue(new Error("only_sso hides every other way in, so an instance Owner needs a passkey first"));
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("switch", { name: "Only single sign-on" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/instance/sso", { only: true });
    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith(expect.stringContaining("needs a passkey first")));
    expect(screen.getByRole("switch", { name: "Only single sign-on" })).not.toBeChecked();
  });

  it("removes the provider after confirming", async () => {
    serve(sso());
    mocks.delete.mockResolvedValue({ data: unset() });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Remove provider" }));
    await user.click(screen.getByRole("button", { name: "Remove" }));

    expect(await screen.findByRole("button", { name: "Set up provider" })).toBeInTheDocument();
    expect(mocks.delete).toHaveBeenCalledWith("/api/instance/sso");
  });
});

describe("ssoFollower", () => {
  it("refetches the settings and the sign-in page's view of them when they change anywhere", async () => {
    const client = seeded([
      [[getSsoKey], sso()],
      [[getSsoPublicKey], { active: true, name: "Acme SSO", only: false }],
    ]);
    await followFrame(ssoFollower, "instance.sso_changed", {}, client);
    expect(isStale(client, [getSsoKey])).toBe(true);
    expect(isStale(client, [getSsoPublicKey])).toBe(true);
  });
});
