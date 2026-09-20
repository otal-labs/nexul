import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { InstanceRecordForm } from "@/components/dns/InstanceRecordForm";
import { pickOption } from "@/test/pickOption";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  errorMessage: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn() },
  onDone: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, post: mocks.post },
  errorMessage: mocks.errorMessage,
}));

vi.mock("sonner", () => ({ toast: mocks.toast }));

const zones = [{ id: "z1", name: "example.com", status: "active" }];
const twoZones = [...zones, { id: "z2", name: "other.dev", status: "active" }];
const settings = { instance_url: "https://deploy.example.com", settings_version: 1 };

// Zones for the zone list, the saved instance URL for the record-name preview.
const mockGets = (zoneList = zones, instanceUrl = settings.instance_url) =>
  mocks.get.mockImplementation(async (url: string) => {
    if (url === "/api/auth/settings") return { data: { ...settings, instance_url: instanceUrl } };
    return { data: zoneList };
  });

const renderForm = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InstanceRecordForm onDone={mocks.onDone} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

const fillAndSubmit = async () => {
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText(/points to/i), "203.0.113.10");
  await user.click(screen.getByRole("button", { name: /create instance record/i }));
  return user;
};

beforeEach(() => {
  mocks.get.mockReset();
  mocks.post.mockReset();
  mocks.errorMessage.mockClear();
  mocks.toast.error.mockClear();
  mocks.onDone.mockClear();
});

describe("InstanceRecordForm", () => {
  it("shows the shared error display when zones fail to load", async () => {
    mocks.get.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("Zones failed");
    renderForm();
    expect(await screen.findByText("Zones failed")).toBeInTheDocument();
  });

  it("preselects the only zone and creates the record with its name filled in", async () => {
    mockGets();
    mocks.post.mockResolvedValue({ data: { id: "r1" } });
    renderForm();
    expect(await screen.findByRole("combobox", { name: /^Zone$/ })).toHaveTextContent("example.com");
    await fillAndSubmit();
    await waitFor(() =>
      expect(mocks.post).toHaveBeenCalledWith("/api/dns/instance-record", {
        zone_id: "z1",
        zone: "example.com",
        type: "A",
        target: "203.0.113.10",
      }),
    );
    expect(mocks.onDone).toHaveBeenCalledWith(expect.objectContaining({ detail: "A → 203.0.113.10" }));
  });

  it("previews the record name derived from the instance URL", async () => {
    mockGets();
    renderForm();
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText(/points to/i), "203.0.113.10");
    expect(await screen.findByText("deploy.example.com → 203.0.113.10")).toBeInTheDocument();
  });

  it("blocks the record when the instance URL is outside the zone, pointing at Settings", async () => {
    mockGets(zones, "http://localhost:5173");
    renderForm();
    const user = userEvent.setup();
    await user.type(await screen.findByLabelText(/points to/i), "203.0.113.10");
    expect(await screen.findByRole("alert")).toHaveTextContent(/localhost:5173.*not under example\.com/i);
    expect(screen.getByRole("link", { name: /settings/i })).toHaveAttribute("href", "/settings?section=instance");
    expect(screen.getByRole("button", { name: /create instance record/i })).toBeDisabled();
  });

  it("leaves the zone unpicked when there is more than one", async () => {
    mockGets(twoZones);
    renderForm();
    expect(await screen.findByRole("combobox", { name: /^Zone$/ })).toHaveTextContent("Choose a zone…");
    const user = userEvent.setup();
    await pickOption(user, /^Zone$/, "other.dev");
    await user.type(screen.getByLabelText(/points to/i), "203.0.113.10");
    expect(screen.getByRole("alert")).toHaveTextContent(/not under other\.dev/i);
    expect(screen.getByRole("button", { name: /create instance record/i })).toBeDisabled();
  });

  it("surfaces the mismatch error via toast, keeps the values, and does not advance", async () => {
    mockGets();
    mocks.post.mockRejectedValue(new Error("boom"));
    mocks.errorMessage.mockReturnValue("hostname is not under zone");
    renderForm();
    await fillAndSubmit();
    await waitFor(() => expect(mocks.toast.error).toHaveBeenCalledWith("hostname is not under zone"));
    expect(screen.getByLabelText(/points to/i)).toHaveValue("203.0.113.10");
    expect(mocks.onDone).not.toHaveBeenCalled();
  });

  it("rejects a submit without a target", async () => {
    mockGets();
    renderForm();
    const user = userEvent.setup();
    await user.click(await screen.findByRole("button", { name: /create instance record/i }));
    expect(await screen.findByText(/server address or tunnel hostname is required/i)).toBeInTheDocument();
    expect(mocks.post).not.toHaveBeenCalled();
  });
});
