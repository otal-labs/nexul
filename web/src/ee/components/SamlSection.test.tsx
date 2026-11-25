import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SsoSection } from "@/ee/components/SsoSection";
import type { SamlSettings } from "@/ee/models/Saml";
import type { SsoSettings } from "@/ee/models/Sso";
import { pickOption } from "@/test/pickOption";
import { savedSaml, unsetSaml } from "@/test/samlSettings";

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

const noOidc = (over: Partial<SsoSettings> = {}): SsoSettings => ({
  unlocked: true,
  configured: false,
  issuer_url: "",
  client_id: "",
  client_secret_set: false,
  display_name: "",
  join_role_id: "",
  only: false,
  redirect_uri: "https://nexul.example.com/auth/oidc/callback",
  active: false,
  ...over,
});

const serve = (saml: SamlSettings, oidc: SsoSettings = noOidc()) =>
  mocks.get.mockImplementation((url: string) =>
    Promise.resolve({ data: ({ "/api/instance/sso": oidc, "/api/instance/saml": saml } as Record<string, unknown>)[url] ?? roles }),
  );

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

describe("SsoSection with SAML", () => {
  it("keeps a saved SAML provider in view without a key, and offers only to remove it", async () => {
    serve(savedSaml({ unlocked: false, active: false }), noOidc({ unlocked: false }));
    renderSection();

    expect(await screen.findByText("Single sign-on needs a license key")).toBeInTheDocument();
    expect(screen.getByText(/saved settings stay/)).toBeInTheDocument();
    expect(screen.getByText("https://idp.example.com")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^(Set up SAML|Set up OpenID Connect|Edit provider)$/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove provider" })).toBeInTheDocument();
  });

  it("asks which protocol at setup, and saves SAML from a metadata URL once both checks pass", async () => {
    serve(unsetSaml());
    mocks.post.mockImplementation((_url: string, _data: unknown, config: { params: { check: string } }) =>
      Promise.resolve({ data: config.params.check === "metadata" ? { detail: "Read https://idp.example.com with 1 signing certificate" } : undefined }),
    );
    mocks.put.mockResolvedValue({ data: savedSaml() });
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByRole("button", { name: "Set up OpenID Connect" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Set up SAML" }));
    expect(screen.getByText(/as entity ID https:\/\/nexul\.example\.com\/auth\/saml\/metadata/)).toBeInTheDocument();
    await user.type(screen.getByLabelText("Metadata URL"), "https://idp.example.com/metadata");
    await user.type(screen.getByLabelText("Name on the button"), "Acme SSO");
    await pickOption(user, "New people join as", "Member");
    expect(screen.getByText(/they can sign in as someone else here/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Verify" }));

    expect(await screen.findByText("Read https://idp.example.com with 1 signing certificate")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/saml/verify", expect.not.objectContaining({ metadata_xml: expect.anything() }), { params: { check: "metadata" } });
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(mocks.put).toHaveBeenCalledWith("/api/instance/saml", {
      metadata_url: "https://idp.example.com/metadata",
      email_attribute: "email",
      name_attribute: "",
      display_name: "Acme SSO",
      join_role_id: "role-member",
    });
  });

  it("sends uploaded metadata as XML read from the field, never with a URL", async () => {
    serve(unsetSaml());
    mocks.post.mockResolvedValue({ data: undefined });
    mocks.put.mockResolvedValue({ data: savedSaml({ metadata_url: "", metadata_xml_set: true }) });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Set up SAML" }));
    await user.click(screen.getByRole("radio", { name: "Metadata XML" }));
    await user.type(screen.getByLabelText("Metadata XML"), "<md:EntityDescriptor/>");
    await user.type(screen.getByLabelText("Name on the button"), "Acme SSO");
    await pickOption(user, "New people join as", "Member");
    await user.click(screen.getByRole("button", { name: "Verify" }));
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalled());
    expect(mocks.put).toHaveBeenCalledWith("/api/instance/saml", expect.objectContaining({ metadata_xml: "<md:EntityDescriptor/>" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/instance/saml", expect.not.objectContaining({ metadata_url: expect.anything() }));
  });

  it("shows the server's refusal when the other protocol was saved meanwhile", async () => {
    serve(unsetSaml());
    mocks.post.mockResolvedValue({ data: undefined });
    mocks.put.mockRejectedValue(new Error("an OpenID Connect provider is saved and an instance holds one provider; remove it first"));
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Set up SAML" }));
    await user.type(screen.getByLabelText("Metadata URL"), "https://idp.example.com/metadata");
    await user.type(screen.getByLabelText("Name on the button"), "Acme SSO");
    await pickOption(user, "New people join as", "Member");
    await user.click(screen.getByRole("button", { name: "Verify" }));
    await user.click(await screen.findByRole("button", { name: "Confirm" }));

    expect(await screen.findByText(/remove it first/)).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("offers no SAML setup while OpenID Connect is saved, and says how to switch", async () => {
    serve(unsetSaml(), noOidc({ configured: true, issuer_url: "https://login.example.com", display_name: "Acme OIDC", active: true }));
    renderSection();

    expect(await screen.findByText(/remove this one to set up SAML instead/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Set up SAML" })).not.toBeInTheDocument();
  });

  it("shows what to register, the metadata failing to refresh, and refreshes it on demand", async () => {
    serve(savedSaml({ metadata_refresh_error: "cannot reach the metadata URL" }));
    mocks.put.mockResolvedValue({ data: savedSaml() });
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("https://nexul.example.com/auth/saml/acs")).toBeInTheDocument();
    expect(screen.getByText(/remove this one to set up OpenID Connect instead/)).toBeInTheDocument();
    expect(screen.getByText("Refresh failing")).toBeInTheDocument();
    expect(screen.getByText("cannot reach the metadata URL")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Refresh" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/instance/saml", { metadata_url: "https://idp.example.com/metadata" });
    expect(await screen.findByText("Read")).toBeInTheDocument();
  });

  it("warns about a certificate within 30 days of expiry, and rotates and retires Nexul's", async () => {
    serve(savedSaml({ certificates: [{ use: "sp", not_after: "2031-11-25T00:00:00Z", expires_soon: false }, { use: "idp", not_after: "2099-01-01T00:00:00Z", expires_soon: true }] }));
    const rotated = savedSaml({
      sp: { ...savedSaml().sp, previous_certificate: "-----BEGIN CERTIFICATE-----old" },
      certificates: [
        { use: "sp", not_after: "2031-11-25T00:00:00Z", expires_soon: false },
        { use: "sp_previous", not_after: "2030-11-25T00:00:00Z", expires_soon: false },
      ],
    });
    mocks.post.mockImplementation((url: string) => Promise.resolve({ data: url.endsWith("/rotate") ? rotated : savedSaml() }));
    const user = userEvent.setup();
    renderSection();

    expect(await screen.findByText("Expires soon")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Rotate" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/saml/rotate");
    expect(await screen.findByText("Nexul's old certificate")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rotate" }), "a second rotation waits for the retirement").not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Retire" }));
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/saml/retire");
    await waitFor(() => expect(screen.queryByText("Nexul's old certificate")).not.toBeInTheDocument());
  });

  it("flips SSO-only through the SAML settings, and removes the SAML provider", async () => {
    serve(savedSaml());
    mocks.put.mockResolvedValue({ data: savedSaml({ only: true }) });
    mocks.delete.mockResolvedValue({ data: unsetSaml() });
    const user = userEvent.setup();
    renderSection();

    await user.click(await screen.findByRole("switch", { name: "Only single sign-on" }));
    expect(mocks.put).toHaveBeenCalledWith("/api/instance/saml", { only: true });

    await user.click(screen.getByRole("button", { name: "Remove provider" }));
    await user.click(screen.getByRole("button", { name: "Remove" }));
    expect(mocks.delete).toHaveBeenCalledWith("/api/instance/saml");
    expect(await screen.findByRole("button", { name: "Set up SAML" })).toBeInTheDocument();
  });
});
