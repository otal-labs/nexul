import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LicenseNotice } from "@/ee/components/LicenseNotice";
import { LicenseSection } from "@/ee/components/LicenseSection";
import { getLicenseKey, licenseFollower } from "@/ee/hooks/LicenseHooks";
import type { License } from "@/ee/models/License";
import { followFrame, isStale, seeded } from "@/test/followFrame";

vi.mock("@/hooks/useActionState", async () => (await import("@/test/liveActions")).liveActions);

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  canChange: true,
  errorMessage: vi.fn((err: unknown) => (err as Error).message),
  toast: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, delete: mocks.delete },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

vi.mock("@/hooks/AccessHooks", () => ({ useHasInstancePermission: () => mocks.canChange }));

const inDays = (days: number) => new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString();

const license = (over: Partial<License> = {}): License => ({
  state: "active",
  licensee: "Acme",
  instance_url: "https://nexul.example.com",
  features: ["sso", "branding"],
  expires_at: inDays(200),
  ...over,
});

const none: License = { state: "none", licensee: "", instance_url: "", features: [] };

const renderWith = (ui: React.ReactNode) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.canChange = true;
});

describe("LicenseSection", () => {
  it("shows a refusal under the key and keeps the dialog open", async () => {
    mocks.get.mockResolvedValue({ data: none });
    mocks.put.mockRejectedValue(new Error("This key is for https://other.example.com, but this instance's address is https://nexul.example.com."));
    renderWith(<LicenseSection />);

    expect(await screen.findByText("This instance runs without enterprise features.")).toBeInTheDocument();
    expect(screen.getByText("No key")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Add license key" }));
    await userEvent.type(screen.getByLabelText("License key"), "nexul-license-1.abc");
    await userEvent.click(screen.getByRole("button", { name: "Save key" }));

    expect(await screen.findByText(/This key is for https:\/\/other.example.com/)).toBeInTheDocument();
    expect(mocks.put).toHaveBeenCalledWith("/api/instance/license", { key: "nexul-license-1.abc" });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("refuses an empty paste without asking the server", async () => {
    mocks.get.mockResolvedValue({ data: none });
    renderWith(<LicenseSection />);
    await userEvent.click(await screen.findByRole("button", { name: "Add license key" }));
    await userEvent.click(screen.getByRole("button", { name: "Save key" }));
    expect(await screen.findByText("Paste the license key")).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("saves a key and shows who it is for, what it unlocks and when it expires", async () => {
    mocks.get.mockResolvedValue({ data: none });
    mocks.put.mockResolvedValue({ data: license({ expires_at: "2027-11-11T00:00:00Z" }) });
    renderWith(<LicenseSection />);

    await userEvent.click(await screen.findByRole("button", { name: "Add license key" }));
    await userEvent.type(screen.getByLabelText("License key"), "nexul-license-1.abc");
    await userEvent.click(screen.getByRole("button", { name: "Save key" }));

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    expect(screen.getByText("Single sign-on")).toBeInTheDocument();
    expect(screen.getByText("Branding")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open Branding" })).toHaveAttribute("href", "/settings/instance#branding");
    expect(screen.getByRole("link", { name: "Open Single sign-on" })).toHaveAttribute("href", "/settings/sign-in#single-sign-on");
    expect(screen.getByText(/11 Nov 2027|Nov 11, 2027/)).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(mocks.toast.success).toHaveBeenCalledWith("License key saved");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("names the days left inside the notice window", async () => {
    mocks.get.mockResolvedValue({ data: license({ state: "expiring", expires_at: inDays(9) }) });
    renderWith(<LicenseSection />);
    expect(await screen.findByText("Expires in 9 days")).toBeInTheDocument();
  });

  it("says the features are off and the data stays once it has expired", async () => {
    mocks.get.mockResolvedValue({ data: license({ state: "expired", expires_at: inDays(-3) }) });
    renderWith(<LicenseSection />);
    expect(await screen.findByText("The enterprise features are off")).toBeInTheDocument();
    expect(screen.getByText("Expired")).toBeInTheDocument();
    expect(screen.getAllByText("Off")).toHaveLength(2);
  });

  it("says why a stored key unlocks nothing here", async () => {
    mocks.get.mockResolvedValue({ data: license({ state: "invalid", problem: "This key is for https://old.example.com." }) });
    renderWith(<LicenseSection />);
    expect(await screen.findByText("This key is for https://old.example.com.")).toBeInTheDocument();
    expect(screen.getByText("Not valid here")).toBeInTheDocument();
  });

  it("removes the key after confirming", async () => {
    mocks.get.mockResolvedValue({ data: license() });
    mocks.delete.mockResolvedValue({ data: none });
    renderWith(<LicenseSection />);

    await userEvent.click(await screen.findByRole("button", { name: "Remove key" }));
    await userEvent.click(screen.getByRole("button", { name: "Remove" }));

    expect(await screen.findByText("This instance runs without enterprise features.")).toBeInTheDocument();
    expect(mocks.delete).toHaveBeenCalledWith("/api/instance/license");
    expect(screen.getByRole("button", { name: "Add license key" })).toBeInTheDocument();
  });

  it("toasts a removal that failed", async () => {
    mocks.get.mockResolvedValue({ data: license() });
    mocks.delete.mockRejectedValue(new Error("boom"));
    renderWith(<LicenseSection />);
    await userEvent.click(await screen.findByRole("button", { name: "Remove key" }));
    await userEvent.click(screen.getByRole("button", { name: "Remove" }));
    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("boom"));
  });
});

describe("LicenseNotice", () => {
  it("tells the Owner the key runs out, with the date", async () => {
    mocks.get.mockResolvedValue({ data: license({ state: "expiring", expires_at: inDays(5) }) });
    renderWith(<LicenseNotice />);
    expect(await screen.findByText(/expires in 5 days/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open license" })).toHaveAttribute("href", "/settings/instance#license");
  });

  it("stays after expiry and says the data is kept", async () => {
    mocks.get.mockResolvedValue({ data: license({ state: "expired", expires_at: inDays(-1) }) });
    renderWith(<LicenseNotice />);
    expect(await screen.findByText(/everything they made is kept/)).toBeInTheDocument();
  });

  it("says a key that no longer fits turned the features off", async () => {
    mocks.get.mockResolvedValue({ data: license({ state: "invalid" }) });
    renderWith(<LicenseNotice />);
    expect(await screen.findByText(/doesn't fit this instance/)).toBeInTheDocument();
  });

  it("is quiet while the key is active", async () => {
    mocks.get.mockResolvedValue({ data: license() });
    renderWith(<LicenseNotice />);
    await waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("never asks for the license for someone who cannot change it", () => {
    mocks.canChange = false;
    renderWith(<LicenseNotice />);
    expect(mocks.get).not.toHaveBeenCalled();
  });
});

describe("licenseFollower", () => {
  it("refetches the license when it changes anywhere", async () => {
    const client = seeded([[[getLicenseKey], none]]);
    await followFrame(licenseFollower, "instance.license_changed", {}, client);
    expect(isStale(client, [getLicenseKey])).toBe(true);
  });
});
