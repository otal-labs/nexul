import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SecretManagersStatus } from "@/components/settings/SecretManagersStatus";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@/api/client", () => ({ api: { get: mocks.get }, errorMessage: vi.fn() }));

const renderStatus = (managers: { manager: string; state: string }[]) => {
  mocks.get.mockResolvedValue({ data: { managers } });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <SecretManagersStatus />
    </QueryClientProvider>,
  );
};

describe("SecretManagersStatus", () => {
  beforeEach(() => mocks.get.mockReset());

  it("names the manager from its scheme and says why it is unusable", async () => {
    renderStatus([{ manager: "op", state: "token rejected" }]);

    const list = await screen.findByRole("list", { name: "Secrets managers" });
    expect(list).toHaveTextContent("1Password");
    expect(list).toHaveTextContent("token rejected");
    expect(mocks.get).toHaveBeenCalledWith("/api/connectors/managers");
  });

  it("shows a ready manager by name alone", async () => {
    renderStatus([{ manager: "op", state: "ready" }]);

    const list = await screen.findByRole("list", { name: "Secrets managers" });
    expect(list).toHaveTextContent("1Password");
    expect(list).not.toHaveTextContent("ready");
  });

  it("shows nothing when the server holds no token", async () => {
    renderStatus([]);

    await vi.waitFor(() => expect(mocks.get).toHaveBeenCalled());
    expect(screen.queryByRole("list", { name: "Secrets managers" })).not.toBeInTheDocument();
  });
});
