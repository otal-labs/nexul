import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { MemoryRouter } from "react-router";
import { toast } from "sonner";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { api } from "@/api/client";
import { DnsSetupDone } from "@/components/dns/DnsSetupDone";
import type { DnsSetupResult } from "@/models/DNS";

const mocks = vi.hoisted(() => ({ canDelete: true }));

vi.mock("@/api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api/client")>()),
  api: { delete: vi.fn() },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/WorkspaceHooks", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/WorkspaceHooks")>()),
  useHasPermission: (value: string) => value !== "dns:delete" || mocks.canDelete,
}));

const result: DnsSetupResult = {
  headline: "The instance record under example.com now points at 203.0.113.10.",
  detail: "A → 203.0.113.10",
  record: { id: "rec-1", zone_id: "zone-1", type: "A", name: "nexul.example.com", content: "203.0.113.10", ttl: 1 },
};

const renderDone = (props: Partial<{ result: DnsSetupResult }> = {}) => {
  const onRecordDeleted = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <MemoryRouter>
        <DnsSetupDone result={props.result ?? result} onRecordDeleted={onRecordDeleted} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onRecordDeleted };
};

describe("DnsSetupDone", () => {
  beforeEach(() => {
    mocks.canDelete = true;
    vi.mocked(api.delete).mockReset().mockResolvedValue({ data: { status: "deleted" } });
    vi.mocked(toast.error).mockClear();
  });

  it("keeps the step finished and shows the server's reason when the delete is refused", async () => {
    vi.mocked(api.delete).mockRejectedValue({ response: { status: 403, data: { message: "forbidden: dns:delete" } } });
    const { onRecordDeleted } = renderDone();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Delete this record" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Delete record" }));

    expect(toast.error).toHaveBeenCalledWith("forbidden: dns:delete");
    expect(onRecordDeleted).not.toHaveBeenCalled();
  });

  it("names the record in the confirm and deletes nothing when cancelled", async () => {
    const { onRecordDeleted } = renderDone();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Delete this record" }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("Delete the A record for nexul.example.com?");
    expect(dialog).toHaveTextContent("stops pointing at 203.0.113.10");
    expect(dialog).toHaveTextContent("can't be undone");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    expect(api.delete).not.toHaveBeenCalled();
    expect(onRecordDeleted).not.toHaveBeenCalled();
  });

  it("deletes the record it created, then sends the step back to the hostname", async () => {
    const { onRecordDeleted } = renderDone();
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Delete this record" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Delete record" }));

    expect(api.delete).toHaveBeenCalledWith("/api/dns/zones/zone-1/records/rec-1");
    await vi.waitFor(() => expect(onRecordDeleted).toHaveBeenCalled());
  });

  it.each([
    ["the viewer cannot delete DNS", false, result],
    ["the step made no record of its own", true, { headline: "Routed", detail: "tunnel" }],
  ])("offers no delete when %s", (_why, canDelete, shown) => {
    mocks.canDelete = canDelete;
    renderDone({ result: shown });

    expect(screen.getByRole("button", { name: "Continue to Nexul" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delete this record" })).not.toBeInTheDocument();
  });
});
