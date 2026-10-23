import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ComputersSection } from "@/components/settings/ComputersSection";
import type { Computer } from "@/models/Pairing";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn(), patch: vi.fn(), post: vi.fn(), del: vi.fn() }));

vi.mock("@/api/client", () => ({
  api: { get: mocks.get, put: mocks.put, patch: mocks.patch, post: mocks.post, delete: mocks.del },
  errorMessage: (e: { message?: string }) => e?.message ?? "Something went wrong",
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("@/components/settings/ComputerMCPToken", () => ({ ComputerMCPToken: () => null }));

const computer = (id: string, name: string, position: number, running = 0): Computer => ({
  id,
  name,
  position,
  runs_at_once: 2,
  running,
  chat_project_id: "",
  server_url: `http://${id}.nexul-computer.invalid`,
  token_expires_at: "2099-01-01T00:00:00Z",
  kind: "t3code",
  harness_version: "0.1.1",
  created_at: "2026-10-01T00:00:00Z",
  updated_at: "2026-10-01T00:00:00Z",
  runner: { connected: true, last_seen: "2026-10-22T12:00:00Z", reconnecting: false },
  facts: { t3: { state: "answering" }, providers: [], projects: [{ id: "t3-notes", title: "notes", path: "/home/alice/notes" }] },
});

let computers: Computer[];

const renderSection = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <ComputersSection />
      </QueryClientProvider>
    </MemoryRouter>,
  );
};

const order = () => within(screen.getByRole("list", { name: /your computers, first runs first/i })).getAllByRole("button", { expanded: false }).map((b) => b.textContent).filter(Boolean);

describe("Your computers", () => {
  beforeEach(() => {
    computers = [computer("c-laptop", "laptop", 0, 1), computer("c-box", "build-box", 1)];
    mocks.get.mockReset().mockImplementation(async (url: string) => {
      if (url === "/api/pairing/computers") return { data: { computers } };
      if (url === "/api/pairing/presence") return { data: { computers: {} } };
      if (url === "/api/pairing/resolve") return { data: { ok: true, computer_id: "c-laptop" } };
      if (url.endsWith("/setup")) return { data: { computer_id: "", confirmed_at: "2026-10-01T00:00:00Z", providers: [], turns: [], skipped_providers: [], models: {}, model_options: {}, folder: "" } };
      return { data: {} };
    });
    mocks.put.mockReset();
    mocks.patch.mockReset();
  });

  it("ranks the computers in their order with how busy each is", async () => {
    renderSection();

    expect(await screen.findByText("#1")).toBeInTheDocument();
    expect(screen.getByText("#2")).toBeInTheDocument();
    expect(await screen.findByText("1 of 2 running")).toBeInTheDocument();
    expect(screen.getByText("0 of 2 running")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reorder laptop" })).toBeInTheDocument();
  });

  it("moves a computer down from its menu, at once, and sends the new order", async () => {
    const user = userEvent.setup();
    mocks.put.mockImplementation(async () => {
      computers = [{ ...computers[1]!, position: 0 }, { ...computers[0]!, position: 1 }];
      return { data: {} };
    });
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for laptop" }));
    await user.click(await screen.findByRole("button", { name: "Move down" }));

    expect(mocks.put).toHaveBeenCalledWith("/api/pairing/computers/order", { computer_ids: ["c-box", "c-laptop"] });
    await waitFor(() => expect(order()).toEqual(["build-box", "laptop"]));
  });

  it("puts the order back when the server refuses it", async () => {
    const user = userEvent.setup();
    mocks.put.mockRejectedValue(new Error("the order must name each of your computers once"));
    renderSection();

    await user.click(await screen.findByRole("button", { name: "Actions for build-box" }));
    await user.click(await screen.findByRole("button", { name: "Move up" }));

    await waitFor(() => expect(order()).toEqual(["laptop", "build-box"]));
  });

  it("sets how many runs a computer takes at once from its details", async () => {
    const user = userEvent.setup();
    mocks.patch.mockImplementation(async (_url: string, body: object) => ({ data: { ...computers[0], ...body } }));
    renderSection();

    await user.click(await screen.findByRole("button", { name: "laptop", expanded: false }));
    await user.click(within(await screen.findByRole("radiogroup", { name: "Runs at once on laptop" })).getByRole("radio", { name: "4" }));

    expect(mocks.patch).toHaveBeenCalledWith("/api/pairing/computers/c-laptop", { runs_at_once: 4 });
    expect(await screen.findByText("1 of 4 running")).toBeInTheDocument();
  });
});
