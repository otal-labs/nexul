import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ContextAwareConfirmation } from "react-confirm";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TunnelsSection } from "@/components/dns/TunnelsSection";
import { api } from "@/api/client";
import { toast } from "sonner";

const mocks = vi.hoisted(() => ({ canDelete: true }));

vi.mock("@/api/client", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/api/client")>()),
  api: { get: vi.fn(), delete: vi.fn() },
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/hooks/WorkspaceHooks", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/WorkspaceHooks")>()),
  useHasPermission: (value: string) => value !== "dns:delete" || mocks.canDelete,
}));

const tunnel = {
  id: "t1",
  name: "edge",
  account_id: "acct",
  status: "active",
  hostname: "nexul.example.com",
  created_at: "",
  updated_at: "",
};

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ContextAwareConfirmation.ConfirmationRoot />
      <TunnelsSection />
    </QueryClientProvider>,
  );
};

describe("TunnelsSection", () => {
  beforeEach(() => {
    mocks.canDelete = true;
    vi.mocked(api.get).mockReset().mockResolvedValue({ data: [tunnel] });
    vi.mocked(api.delete).mockReset();
    vi.mocked(toast.error).mockClear();
  });

  it("names what goes with the tunnel in the confirm, and deletes nothing when it is cancelled", async () => {
    renderSection();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Delete tunnel edge" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/The tunnel/)).toHaveTextContent("The tunnel edge at Cloudflare");
    expect(within(dialog).getByText(/Its route/)).toHaveTextContent("nexul.example.com");
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

    expect(api.delete).not.toHaveBeenCalled();
  });

  it("shows the refusal that names the hostnames and gateways using the tunnel", async () => {
    const refusal = 'tunnel "edge" is still used by app.example.com, gateway on network web; edit or remove them first';
    vi.mocked(api.delete).mockRejectedValue({ response: { status: 409, data: { message: refusal, code: "CONFLICT" } } });
    renderSection();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Delete tunnel edge" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Delete tunnel" }));

    expect(api.delete).toHaveBeenCalledWith("/api/dns/tunnels/t1");
    expect(toast.error).toHaveBeenCalledWith(refusal);
  });

  it("hides Delete from someone who can't delete DNS", async () => {
    mocks.canDelete = false;
    renderSection();

    expect(await screen.findByText("edge")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delete tunnel edge" })).not.toBeInTheDocument();
  });
});
