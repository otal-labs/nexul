import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuditLogSection } from "@/ee/components/AuditLogSection";
import type { AuditSettings } from "@/ee/models/AuditLog";

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

const settings = (over: Partial<AuditSettings> = {}): AuditSettings => ({
  unlocked: true,
  retention_days: 45,
  stream_url: "",
  stream_secret_set: false,
  stream_enabled: false,
  streaming: false,
  pending: 0,
  dead: 0,
  ...over,
});

const streaming = (over: Partial<AuditSettings> = {}) =>
  settings({ retention_days: 365, stream_url: "https://logs.example.com/nexul", stream_secret_set: true, stream_enabled: true, streaming: true, ...over });

const serve = (s: AuditSettings) => mocks.get.mockResolvedValue({ data: s });

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AuditLogSection />
      </MemoryRouter>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("AuditLogSection", () => {
  it("with the key locked, says the saved days still hold, keeps the form read-only and offers only Remove", async () => {
    serve(streaming({ unlocked: false, streaming: false }));
    renderSection();

    expect(await screen.findByText("Keeping the log longer and streaming it need a license key")).toBeInTheDocument();
    expect(screen.getByText(/still kept 365 days/)).toBeInTheDocument();
    expect(screen.getByText("Not streaming")).toBeInTheDocument();
    expect(screen.getByLabelText("Days kept")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Send a test" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Remove settings" })).toBeInTheDocument();
  });

  it("not set up, shows 45 days, no deliveries and nothing to remove", async () => {
    serve(settings());
    renderSection();

    expect(await screen.findByLabelText("Days kept")).toHaveValue("45");
    expect(screen.getByText("kept 45 days")).toBeInTheDocument();
    expect(screen.queryByText("Deliveries")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove settings" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rotate secret" })).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText("Leave blank to have one made")).toBeInTheDocument();
  });

  it("with a failing destination, shows the failed rows and the reason, and Retry failed queues them again", async () => {
    serve(streaming({ dead: 3, pending: 1, last_error: "the destination answered 503" }));
    mocks.post.mockResolvedValue({ data: streaming({ pending: 4 }) });
    renderSection();

    expect(await screen.findByText("3 failed")).toBeInTheDocument();
    expect(screen.getByText("the destination answered 503")).toBeInTheDocument();
    expect(screen.getByText("1 waiting")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Retry failed" }));

    await waitFor(() => expect(mocks.post).toHaveBeenCalledWith("/api/instance/audit/stream/retry"));
    expect(await screen.findByText("4 waiting")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retry failed" })).not.toBeInTheDocument();
  });

  it("with a delivery pending and none sent yet, says both", async () => {
    serve(streaming({ pending: 2 }));
    renderSection();
    expect(await screen.findByText("2 waiting")).toBeInTheDocument();
    expect(screen.getByText("nothing sent yet")).toBeInTheDocument();
  });

  it("shows a secret the server made once, and not after the next read", async () => {
    serve(settings());
    mocks.put.mockResolvedValue({ data: streaming({ stream_enabled: false, streaming: false, new_secret: "f00dfeed" }) });
    renderSection();

    await userEvent.type(await screen.findByLabelText("Destination"), "https://logs.example.com/nexul");
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.put).toHaveBeenCalledWith("/api/instance/audit", { retention_days: 45, stream_url: "https://logs.example.com/nexul", stream_enabled: false }));
    expect(await screen.findByText("f00dfeed")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Saved; type to replace it")).toBeInTheDocument();
  });

  it("refuses days out of range and switching on with no destination before saving", async () => {
    serve(settings());
    renderSection();

    const days = await screen.findByLabelText("Days kept");
    await userEvent.clear(days);
    await userEvent.type(days, "30");
    await userEvent.click(screen.getByRole("switch"));
    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText("From 45 to 1,095 days")).toBeInTheDocument();
    expect(screen.getByText("Add a destination before sending")).toBeInTheDocument();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("sends a test to the typed destination and shows what the receiver answered", async () => {
    serve(streaming());
    mocks.post.mockResolvedValueOnce({ data: { status: 401, error: "the destination answered 401" } }).mockResolvedValueOnce({ data: { status: 204 } });
    renderSection();

    await userEvent.type(await screen.findByLabelText("Signing secret"), "typed");
    await userEvent.click(screen.getByRole("button", { name: "Send a test" }));
    expect(await screen.findByText("Not taken")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/audit/stream/test", { stream_url: "https://logs.example.com/nexul", stream_secret: "typed" });

    await userEvent.click(screen.getByRole("button", { name: "Send a test" }));
    expect(await screen.findByText("Received")).toBeInTheDocument();
    expect(screen.getByText("answered 204")).toBeInTheDocument();
  });

  it("rotates the secret and shows the new one once", async () => {
    serve(streaming());
    mocks.post.mockResolvedValue({ data: streaming({ new_secret: "c0ffee" }) });
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Rotate secret" }));

    expect(await screen.findByText("c0ffee")).toBeInTheDocument();
    expect(mocks.post).toHaveBeenCalledWith("/api/instance/audit/stream/rotate-secret");
  });

  it("removes the settings after a confirm", async () => {
    serve(streaming());
    mocks.delete.mockResolvedValue({ data: settings() });
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Remove settings" }));
    await userEvent.click(screen.getByRole("button", { name: "Remove" }));

    await waitFor(() => expect(mocks.delete).toHaveBeenCalledWith("/api/instance/audit"));
    expect(mocks.toast.success).toHaveBeenCalledWith("Audit log settings removed");
    expect(await screen.findByText("kept 45 days")).toBeInTheDocument();
  });
});
